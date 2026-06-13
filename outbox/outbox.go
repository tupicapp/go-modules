// Package outbox implements the Transactional Outbox pattern: events are stored in the caller's DB transaction and a
// relay ships them to NATS.
package outbox

import "context"

// OutboxEvent is the constraint for publishable integration events.
type OutboxEvent interface {
	Subject() string
	Version() string
}

// Outbox stores integration events to the outbox for at-least-once delivery to the central message bus.
type Outbox interface {
	Store(ctx context.Context, e OutboxEvent) error
}

// Config carries the relay's publishing identity. SubjectPrefix prefixes every published subject; Source identifies the
// publishing service in the event envelope (typically the app slug).
type Config struct {
	SubjectPrefix string
	Source        string
}
