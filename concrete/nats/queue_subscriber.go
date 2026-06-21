package nats

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	natslib "github.com/nats-io/nats.go"
	msgrouter "github.com/tupicapp/go-modules/concrete/messaging_router"
	"github.com/tupicapp/go-modules/concrete/sentry"
	"github.com/tupicapp/go-modules/contract/clock"
	"github.com/tupicapp/go-modules/contract/logger"
	messaging2 "github.com/tupicapp/go-modules/contract/messaging"
	"github.com/tupicapp/go-modules/shared/apperror"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	// defaultMaxDeliver is total delivery attempts (1 original + 10 retries). Must equal len(backoffSchedule)+1 so the
	// final attempt triggers DLQ.
	defaultMaxDeliver = 11

	// defaultAckWait is how long JetStream waits for any response (Ack/Nak) before treating the handler as stuck and
	// redelivering. Must exceed the slowest expected handler execution time.
	defaultAckWait = 30 * time.Second
)

// backoffSchedule defines the wait before each retry. Spreads 10 retries over ~5.5 days so a multi-day downstream
// outage does not immediately DLQ.
//
// NOTE: this only applies when the service is UP but handlers are failing. If the service process is completely down,
// messages wait undelivered in the JetStream stream — stream MaxAge is what prevents loss in that case.
var backoffSchedule = []time.Duration{
	5 * time.Second,
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	6 * time.Hour,
	12 * time.Hour,
	24 * time.Hour,
	48 * time.Hour,
	72 * time.Hour,
}

// QueueSubscriber subscribes to queues.* subjects and dispatches each task to the Router. Terminal failures (apperror)
// and exhausted MaxDeliver are written to the failed_messages DLQ and Term'd; transient failures are Nak'd for
// JetStream retry.
type QueueSubscriber struct {
	logger    logger.Logger
	clock     clock.Clock
	js        natslib.JetStreamContext
	router    *msgrouter.Router
	repo      *failedMessageRepository
	prefix    string
	appSlug   string
	subs      []*natslib.Subscription
	wg        sync.WaitGroup
	cancelCtx context.Context //nolint:containedctx // lifetime context, not request-scoped
	cancel    context.CancelFunc
}

func NewQueueSubscriber(
	l logger.Logger,
	c clock.Clock,
	js natslib.JetStreamContext,
	router *msgrouter.Router,
	cfg Config,
	db *gorm.DB,
) (*QueueSubscriber, error) {
	if cfg.SubjectPrefix == "" {
		return nil, errors.New("nats worker: nats.subject_prefix is required")
	}
	ctx, cancel := newLifecycleContext()
	return &QueueSubscriber{
		logger:    l,
		clock:     c,
		js:        js,
		router:    router,
		repo:      newFailedMessageRepository(db),
		prefix:    cfg.SubjectPrefix,
		appSlug:   cfg.AppSlug,
		cancelCtx: ctx,
		cancel:    cancel,
	}, nil
}

// Start subscribes to every registered queues.* subject. Each subject gets its own durable consumer so JetStream
// load-balances across replicas naturally.
func (w *QueueSubscriber) Start(_ context.Context) error {
	for _, subject := range w.router.Subjects() {
		if !strings.HasPrefix(subject, "queues.") {
			continue
		}
		wireSubject := w.prefix + "." + subject
		durable := durableNameFor(w.appSlug, subject)
		sub, err := w.js.Subscribe(
			wireSubject,
			w.handleMessage,
			natslib.Durable(durable),
			natslib.ManualAck(),
			natslib.MaxDeliver(defaultMaxDeliver),
			natslib.AckWait(defaultAckWait),
		)
		if err != nil {
			return errors.Wrapf(err, "nats worker: subscribe to %s", wireSubject)
		}
		w.subs = append(w.subs, sub)
		w.logger.Info("nats worker: subscribed",
			zap.String("subject", wireSubject),
			zap.String("durable", durable),
			zap.Int("max_deliver", defaultMaxDeliver),
		)
	}
	return nil
}

// Stop cancels the handler context, unsubscribes all consumers (collecting any errors), then waits for in-flight
// handlers to drain. All subscriptions are unsubscribed regardless of individual errors. If ctx expires before the
// drain completes a warning is logged so downstream lifecycle hooks are not blocked.
func (w *QueueSubscriber) Stop(ctx context.Context) error {
	w.cancel()

	var unsubErrs []error
	for _, sub := range w.subs {
		if err := sub.Unsubscribe(); err != nil {
			unsubErrs = append(unsubErrs, errors.Wrap(err, "nats worker: unsubscribe"))
		}
	}

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("nats worker: stopped")
	case <-ctx.Done():
		w.logger.Warn("nats worker: stop timed out; in-flight handlers may still be running")
	}

	return errors.Join(unsubErrs...)
}

func (w *QueueSubscriber) handleMessage(msg *natslib.Msg) {
	w.wg.Add(1)
	defer w.wg.Done()

	subject := strings.TrimPrefix(msg.Subject, w.prefix+".")

	meta, metaErr := msg.Metadata()
	fields := []zap.Field{
		zap.String("subject", msg.Subject),
		zap.String("task_type", subject),
	}

	// If metadata is unavailable we cannot determine delivery count. Treat as terminal to guarantee a DLQ row rather than
	// silently losing the message when JetStream exhausts MaxDeliver at the broker level.
	if metaErr != nil {
		w.logger.Error("nats worker: metadata unavailable, quarantining",
			append(fields, zap.Error(metaErr))...)
		var env envelope
		if jsonErr := json.Unmarshal(msg.Data, &env); jsonErr == nil {
			w.deadLetter(&env, subject, 0, metaErr)
		}
		w.term(msg, fields)
		return
	}

	numDelivered := meta.NumDelivered
	fields = append(fields, zap.Uint64("num_delivered", numDelivered))

	var env envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		w.logger.Error("nats worker: malformed envelope", append(fields, zap.Error(err))...)
		w.deadLetter(&env, subject, int(numDelivered), err)
		w.term(msg, fields)
		return
	}

	fields = append(fields, zap.String("task_id", env.ID))

	m := messaging2.Message{Version: env.Version, Payload: env.Data}
	err := w.router.Handle(w.cancelCtx, subject, m)
	if err == nil {
		w.logger.Info("nats worker: task handled", fields...)
		w.ack(msg, fields)
		return
	}

	// Terminal (apperror): give up immediately.
	if apperror.IsAppError(err) {
		w.logger.Warn("nats worker: task rejected (terminal)", append(fields, zap.Error(err))...)
		w.deadLetter(&env, subject, int(numDelivered), err)
		w.term(msg, fields)
		return
	}

	// Retriable but last attempt — DLQ and Term.
	if numDelivered >= defaultMaxDeliver {
		w.logger.Error("nats worker: task exhausted, moved to dead-letter",
			append(fields, zap.Error(err))...)
		sentry.Capture(err, map[string]string{"transport": "nats", "task_type": subject})
		w.deadLetter(&env, subject, int(numDelivered), err)
		w.term(msg, fields)
		return
	}

	// Retriable with attempts remaining — Nak with exponential backoff.
	delay := backoffSchedule[min(int(numDelivered)-1, len(backoffSchedule)-1)]
	w.logger.Error("nats worker: task failed (will retry)",
		append(fields, zap.Error(err), zap.Duration("retry_in", delay))...)
	sentry.Capture(err, map[string]string{"transport": "nats", "task_type": subject})
	if nakErr := msg.NakWithDelay(delay); nakErr != nil {
		w.logger.Error("nats worker: nak failed", append(fields, zap.Error(nakErr))...)
	}
}

func (w *QueueSubscriber) ack(msg *natslib.Msg, fields []zap.Field) {
	if err := msg.Ack(); err != nil {
		w.logger.Error("nats worker: ack failed", append(fields, zap.Error(err))...)
	}
}

func (w *QueueSubscriber) term(msg *natslib.Msg, fields []zap.Field) {
	if err := msg.Term(); err != nil {
		w.logger.Error("nats worker: term failed", append(fields, zap.Error(err))...)
	}
}

// deadLetter writes the failed task to the DLQ table. Failures here are logged but not propagated — losing the DLQ row
// is worse than continuing, and JetStream MaxDeliver guarantees the message has already been retried the allowed number
// of times.
func (w *QueueSubscriber) deadLetter(e *envelope, subject string, attempts int, handlerErr error) {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		// Envelope ID was unparseable; mint a fresh one so we don't lose the DLQ entry. Should never happen since the outbox
		// writes UUIDv7s.
		w.logger.Error("nats worker: dlq id unparseable; minting random",
			zap.String("envelope_id", e.ID), zap.Error(err))
		id = uuid.New()
	}
	// Malformed envelopes have no payload; the column is NOT NULL, so store an explicit JSON null instead of losing the
	// DLQ row entirely.
	payload := e.Data
	if len(payload) == 0 {
		payload = json.RawMessage("null")
	}
	row := &FailedMessage{
		ID:        id,
		Type:      subject,
		Version:   e.Version,
		Payload:   datatypes.JSON(payload),
		Attempts:  attempts,
		LastError: handlerErr.Error(),
		FailedAt:  w.clock.Now().UTC(),
	}
	if err := w.repo.save(context.Background(), row); err != nil {
		w.logger.Error("nats worker: failed to write dlq row",
			zap.String("task_id", e.ID),
			zap.Error(err),
		)
	}
}
