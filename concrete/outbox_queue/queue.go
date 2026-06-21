// Package outbox_queue implements the queue.Queue contract on top of the outbox:
// each task is stored as an integration event whose subject carries the channel,
// so the relay ships it to the work-queue stream.
package outbox_queue

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/contract/outbox"
	"github.com/tupicapp/go-modules/contract/queue"
)

// OutboxQueue implements queue.Queue by reusing the outbox publisher.
type OutboxQueue struct {
	publisher outbox.Outbox
}

func NewOutboxQueue(publisher outbox.Outbox) *OutboxQueue {
	return &OutboxQueue{publisher: publisher}
}

// Enqueue stages a task on the default channel. MUST be called inside a UnitOfWork
// transaction so the outbox row write is atomic with the triggering state change.
func (q *OutboxQueue) Enqueue(ctx context.Context, t queue.Task) error {
	return q.EnqueueOn(ctx, queue.DefaultChannel, t)
}

// EnqueueOn stages a task on a named channel.
func (q *OutboxQueue) EnqueueOn(ctx context.Context, channel string, t queue.Task) error {
	return errors.WithStack(q.publisher.Store(ctx, taskAsEvent{task: t, channel: channel}))
}

// taskAsEvent adapts a Task to the outbox.IntegrationEvent shape, composing the
// wire subject from the channel and the task name. MarshalJSON delegates to the
// task so the stored payload is the task's own JSON.
type taskAsEvent struct {
	task    queue.Task
	channel string
}

func (e taskAsEvent) Subject() string              { return queue.Subject(e.channel, e.task.Name()) }
func (e taskAsEvent) Version() string              { return e.task.Version() }
func (e taskAsEvent) MarshalJSON() ([]byte, error) { return json.Marshal(e.task) }

var _ queue.Queue = (*OutboxQueue)(nil)
