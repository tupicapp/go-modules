package outbox

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	natslib "github.com/nats-io/nats.go"
	"github.com/tupicapp/common-go/clock"
	"github.com/tupicapp/common-go/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// defaultPollInterval is how long the relay waits when the outbox is empty before polling again. Kept low so rows
	// written inside a UoW transaction (where the storage write cannot signal after commit) are picked up quickly.
	defaultPollInterval = 100 * time.Millisecond

	// defaultErrorBackoff is the wait after a broker or DB failure before retrying. Avoids hammering a broken system while
	// still recovering fast.
	defaultErrorBackoff = 5 * time.Second

	defaultBatchSize = 100
)

// drainResult tells the loop how long to wait before the next drain.
type drainResult uint8

const (
	drainIdle   drainResult = iota // outbox empty — wait pollInterval
	drainBusy                      // full batch processed — more likely queued
	drainFailed                    // broker/DB error — wait errorBackoff
)

type envelope struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	Source    string          `json:"source"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// Relay polls the outbox for unpublished rows, publishes each to NATS JetStream, then marks the row published. It runs
// as a single goroutine; no per-row concurrency keeps order-per-subject roughly stable.
//
// Loop behavior:
//   - Empty outbox → sleep pollInterval, then poll again.
//   - Full batch → loop back immediately (more events likely queued).
//   - Broker/DB error → sleep errorBackoff, then retry.
//
// Two failure modes for individual rows:
//   - Permanent (marshal failure): quarantined immediately, does not block others.
//   - Transient (NATS unavailable): batch stopped, retried after errorBackoff.
type Relay struct {
	logger       logger.Logger
	clock        clock.Clock
	repo         *repository
	js           natslib.JetStreamContext
	prefix       string
	source       string
	pollInterval time.Duration
	errorBackoff time.Duration
	batchSize    int

	ctx     context.Context //nolint:containedctx // lifetime context, not request-scoped
	cancel  context.CancelFunc
	done    chan struct{}
	started atomic.Bool
}

func NewRelay(
	l logger.Logger,
	c clock.Clock,
	db *gorm.DB,
	js natslib.JetStreamContext,
	cfg Config,
) (*Relay, error) {
	if cfg.SubjectPrefix == "" {
		return nil, errors.New("nats.subject_prefix is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Relay{
		logger:       l,
		clock:        c,
		repo:         newRepository(db),
		js:           js,
		prefix:       cfg.SubjectPrefix,
		source:       cfg.Source,
		pollInterval: defaultPollInterval,
		errorBackoff: defaultErrorBackoff,
		batchSize:    defaultBatchSize,
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
	}, nil
}

// Start kicks off the polling goroutine. The startup ctx is checked for early cancellation only — the loop uses the
// relay's own lifetime context so it is not bounded by the short-lived fx startup context. Idempotent.
func (r *Relay) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.WithStack(ctx.Err())
	}
	if !r.started.CompareAndSwap(false, true) {
		return nil
	}
	go r.loop()
	r.logger.Info("outbox: relay started",
		zap.Duration("poll_interval", r.pollInterval),
		zap.Duration("error_backoff", r.errorBackoff),
		zap.Int("batch_size", r.batchSize),
	)
	return nil
}

// Stop cancels the relay's context, causing the loop to exit, then waits for the goroutine to finish.
func (r *Relay) Stop(ctx context.Context) error {
	r.cancel()
	select {
	case <-r.done:
		r.logger.Info("outbox: relay stopped")
		return nil
	case <-ctx.Done():
		return errors.WithStack(ctx.Err())
	}
}

func (r *Relay) loop() {
	defer close(r.done)
	for {
		switch r.drain() {
		case drainBusy:
			// Full batch — more events likely queued. Yield but do not sleep.
			select {
			case <-r.ctx.Done():
				return
			default:
			}
		case drainIdle:
			// Outbox empty — sleep until the next poll interval.
			select {
			case <-r.ctx.Done():
				return
			case <-time.After(r.pollInterval):
			}
		case drainFailed:
			// Broker or DB error — back off before retrying.
			select {
			case <-r.ctx.Done():
				return
			case <-time.After(r.errorBackoff):
			}
		}
	}
}

func (r *Relay) drain() drainResult {
	events, err := r.repo.listUnpublished(r.ctx, r.batchSize)
	if err != nil {
		r.logger.Error("cannot list unpublished events", zap.Error(err))
		return drainFailed
	}
	if len(events) == 0 {
		return drainIdle
	}

	for _, e := range events {
		eventEnvelope, err := r.makeEnvelope(e)
		if err != nil {
			// Permanent: malformed payload will never publish. Quarantine immediately so it does not block other events.
			r.logger.Error("quarantining event (marshal failure)",
				zap.String("message_id", e.MessageID.String()),
				zap.String("subject", e.Subject),
				zap.Error(err),
			)
			if err := r.repo.quarantine(r.ctx, e.MessageID, err.Error(), r.clock.Now().UTC()); err != nil {
				r.logger.Error("quarantine failed",
					zap.String("message_id", e.MessageID.String()),
					zap.Error(err),
				)
			}

			continue
		}

		subject := r.prefix + "." + e.Subject
		if _, err := r.js.Publish(subject, eventEnvelope, natslib.MsgId(e.MessageID.String())); err != nil {
			// Transient: NATS unavailable. Stop the batch — all pending events retry after errorBackoff. Never quarantine for
			// NATS errors.
			r.logger.Error("outbox: NATS unavailable, stopping batch",
				zap.String("message_id", e.MessageID.String()),
				zap.String("subject", subject),
				zap.Error(err),
			)
			return drainFailed
		}

		if err := r.repo.markPublished(r.ctx, e.MessageID, r.clock.Now().UTC()); err != nil {
			// Already published to NATS. JetStream dedupes by MsgId within its Duplicates window — configure the stream with a
			// window >= your maximum expected relay downtime to avoid duplicate delivery.
			r.logger.Error("outbox: mark published failed",
				zap.String("message_id", e.MessageID.String()),
				zap.Error(err),
			)
			continue
		}

		r.logger.Info("outbox: event relayed",
			zap.String("message_id", e.MessageID.String()),
			zap.String("subject", subject),
		)
	}

	// Full batch: more events may be queued. Partial: outbox is clear.
	if len(events) == r.batchSize {
		return drainBusy
	}
	return drainIdle
}

func (r *Relay) makeEnvelope(e *Event) ([]byte, error) {
	data, err := json.Marshal(envelope{
		ID:        e.MessageID.String(),
		Version:   e.Version,
		Source:    r.source,
		Timestamp: e.OccurredAt.UTC().UnixMilli(),
		Data:      json.RawMessage(e.Payload),
	})
	return data, errors.WithStack(err)
}
