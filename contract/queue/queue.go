// Package queue defines the task queue contract. the Queue port,
// and the channel→subject mapping. The outbox-backed implementation lives in
// adapters/outbox_queue.
package queue

import "context"

// DefaultChannel is the channel used by Enqueue. Callers that do not care about channels and prioritization never name
// a channel; the transport layer routes them here.
const DefaultChannel = "default"

// Queue enqueues tasks for asynchronous, single-consumer execution. Enqueue uses
// the default channel; EnqueueOn routes onto a named channel (e.g. a priority lane).
type Queue interface {
	Enqueue(ctx context.Context, t Task) error
	EnqueueOn(ctx context.Context, channel string, t Task) error
}

// Subject composes the wire message queue subject for a task on a channel: queues.<channel>.<name>.
func Subject(channel, name string) string {
	return "queues." + channel + "." + name
}
