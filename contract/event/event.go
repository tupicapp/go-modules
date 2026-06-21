// Package event defines the domain-event contract and the in-process synchronous event bus shared by all platform
// services.
package event

import "context"

// DomainEvent is the contract every aggregate-raised event implements.
type DomainEvent interface {
	Name() string
}

// Handler reacts to a single domain event.
type Handler func(ctx context.Context, e DomainEvent) error

// Publisher dispatches domain events to the in-process sync event bus.
type Publisher interface {
	Publish(ctx context.Context, e DomainEvent) error
	PublishAll(ctx context.Context, events []DomainEvent) error
}

// Subscriber registers domain event handlers on the in-process sync event bus.
type Subscriber interface {
	Subscribe(eventName string, h Handler)
}
