package event

import (
	"context"

	"github.com/cockroachdb/errors"
)

// TypedHandler reacts to a single concrete domain-event type E. It is the typed
// counterpart of Handler, which receives the erased DomainEvent interface.
type TypedHandler[E DomainEvent] func(ctx context.Context, e E) error

// On registers a typed handler with the subscriber. The routing key is taken
// from the event type itself, so the call site carries no string and the wiring
// is compiler-checked: a handler for E can only ever be registered under E's own
// name. Go does not allow generic methods, so this is a free function over the
// Subscriber rather than a method on it.
//
//	event.On(sub, assetHandler.HandleAssetCreated) // E inferred as *asset.Created
//
// E's Name must be safe to call on its zero value — it must return a constant
// and never dereference the receiver — because On reads the routing key from a
// zero E at registration time (no reflection). A type that violates this fails
// fast at registration with a clear error rather than a cryptic nil panic.
func On[E DomainEvent](s Subscriber, h func(ctx context.Context, e E) error) {
	s.Subscribe(nameOf[E](), func(ctx context.Context, e DomainEvent) error {
		// Safe assertion: the bus only routes E's name to this closure.
		return h(ctx, e.(E))
	})
}

// nameOf returns the routing key for event type E by reading Name from a zero
// value. A panic (e.g. a Name that dereferences a nil pointer receiver) is
// rewritten into an actionable registration error.
func nameOf[E DomainEvent]() (name string) {
	var zero E
	defer func() {
		if r := recover(); r != nil {
			panic(errors.Errorf(
				"event: cannot read routing key from zero %T — Name() must not dereference its receiver: %v",
				zero, r,
			))
		}
	}()
	return zero.Name()
}
