// Package queue defines the platform work-queue contract and its outbox-backed implementation.
package queue

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/outbox"
)

// Task is the constraint for queueable point-to-point work units. Each task type declares its own subject and schema
// version. Tasks are imperative ("validate-asset"), single-consumer, and live on the work-queue stream.
type Task interface {
	Subject() string
	Version() string
}

// Queue enqueues tasks for asynchronous execution.
type Queue interface {
	Enqueue(ctx context.Context, t Task) error
}

// OutboxQueue implements Queue by reusing the outbox publisher: a Task has the same Subject()/Version() shape as an
// OutboxEvent, so the outbox row layout already fits. The relay routes by subject (events.* vs queue.*) to the
// appropriate JetStream stream.
type OutboxQueue struct {
	publisher outbox.Outbox
}

func NewOutboxQueue(publisher outbox.Outbox) *OutboxQueue {
	return &OutboxQueue{publisher: publisher}
}

// Enqueue stages a task for at-least-once, single-consumer execution. MUST be called inside a UnitOfWork transaction so
// the outbox row write is atomic with the aggregate state change that triggered the enqueue.
func (q *OutboxQueue) Enqueue(ctx context.Context, t Task) error {
	return errors.WithStack(q.publisher.Store(ctx, taskAsEvent{t}))
}

// taskAsEvent adapts a Task to the OutboxEvent shape. Both interfaces have identical methods; the wrapper is structural
// plumbing only.
type taskAsEvent struct{ Task }

var _ Queue = (*OutboxQueue)(nil)
