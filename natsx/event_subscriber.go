package natsx

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	natslib "github.com/nats-io/nats.go"
	"github.com/tupic/common-go/apperror"
	"github.com/tupic/common-go/logger"
	"go.uber.org/zap"
)

// defaultEventAckWait is how long NATS waits for an Ack on an event consumer
// before redelivering. Must exceed the slowest expected handler execution.
const defaultEventAckWait = 30 * time.Second

type envelope struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	Source    string          `json:"source"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// EventSubscriber handles non-queue subjects (integration events from other services).
type EventSubscriber struct {
	logger    logger.Logger
	js        natslib.JetStreamContext
	handler   *Router
	appSlug   string
	subs      []*natslib.Subscription
	wg        sync.WaitGroup
	cancelCtx context.Context //nolint:containedctx // lifetime context, not request-scoped
	cancel    context.CancelFunc
}

func NewEventSubscriber(
	l logger.Logger,
	js natslib.JetStreamContext,
	handler *Router,
	cfg Config,
) *EventSubscriber {
	ctx, cancel := newLifecycleContext()
	return &EventSubscriber{
		logger:    l,
		js:        js,
		handler:   handler,
		appSlug:   cfg.AppSlug,
		cancelCtx: ctx,
		cancel:    cancel,
	}
}

func (s *EventSubscriber) Start(_ context.Context) error {
	for _, subject := range s.handler.Subjects() {
		if strings.HasPrefix(subject, "queue.") {
			continue
		}
		durable := durableNameFor(s.appSlug, subject)
		sub, err := s.js.Subscribe(
			subject,
			s.handleMessage,
			natslib.Durable(durable),
			natslib.ManualAck(),
			natslib.AckWait(defaultEventAckWait),
		)
		if err != nil {
			return errors.Wrapf(err, "nats: subscribe to %s", subject)
		}

		s.subs = append(s.subs, sub)
		s.logger.Info("nats: subscribed",
			zap.String("subject", subject),
			zap.String("durable", durable),
		)
	}

	return nil
}

// Stop cancels the handler context, unsubscribes all consumers (collecting
// any errors), then waits for in-flight handlers to drain. All subscriptions
// are unsubscribed regardless of individual errors. If ctx expires before the
// drain completes a warning is logged and the function returns so downstream
// lifecycle hooks are not blocked.
func (s *EventSubscriber) Stop(ctx context.Context) error {
	s.cancel()

	var unsubErrs []error
	for _, sub := range s.subs {
		if err := sub.Unsubscribe(); err != nil {
			unsubErrs = append(unsubErrs, errors.Wrap(err, "nats: unsubscribe"))
		}
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("nats: subscriber stopped")
	case <-ctx.Done():
		s.logger.Warn("nats: subscriber stop timed out; in-flight handlers may still be running")
	}

	return errors.Join(unsubErrs...)
}

func (s *EventSubscriber) handleMessage(msg *natslib.Msg) {
	s.wg.Add(1)
	defer s.wg.Done()

	fields := []zap.Field{
		zap.String("subject", msg.Subject),
		zap.Int("data_size", len(msg.Data)),
	}
	s.logger.Info("nats: message received", fields...)

	var env envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		s.logger.Error("nats: malformed envelope", append(fields, zap.Error(err))...)
		if termErr := msg.Term(); termErr != nil {
			s.logger.Error("nats: failed to terminate message", append(fields, zap.Error(termErr))...)
		}
		return
	}

	s.logger.Info("nats: event received",
		zap.String("subject", msg.Subject),
		zap.String("event_id", env.ID),
		zap.String("source", env.Source),
	)

	m := Message{Version: env.Version, Payload: env.Data}
	if err := s.handler.Handle(s.cancelCtx, msg.Subject, m); err != nil {
		if apperror.IsAppError(err) {
			s.logger.Warn("nats: message rejected (terminal)", append(fields, zap.Error(err))...)
			if termErr := msg.Term(); termErr != nil {
				s.logger.Error("nats: failed to terminate message", append(fields, zap.Error(termErr))...)
			}
		} else {
			s.logger.Error("nats: message failed (will retry)", append(fields, zap.Error(err))...)
			if nakErr := msg.Nak(); nakErr != nil {
				s.logger.Error("nats: failed to nak message", append(fields, zap.Error(nakErr))...)
			}
		}
		return
	}

	s.logger.Info("nats: message handled", fields...)
	if ackErr := msg.Ack(); ackErr != nil {
		s.logger.Error("nats: failed to ack message", append(fields, zap.Error(ackErr))...)
	}
}
