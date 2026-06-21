// Package outbox implements the Transactional Outbox pattern: events are stored in the caller's DB transaction and a
// relay ships them to NATS asynchronously.
package outbox

import "context"

// IntegrationEvent is the constraint for publishable integration events.
type IntegrationEvent interface {
	Subject() string
	Version() string
}

// Outbox stores integration events to the outbox for at-least-once delivery to the central message bus.
type Outbox interface {
	Store(ctx context.Context, e IntegrationEvent) error
}
