// Package messaging defines the transport-agnostic inbound message contract and
// router shared by subscription adapters (NATS, SQS, …). Adapters unwrap their
// own wire envelope into a messaging.Message and dispatch it through a Router;
// application handlers depend only on this package, never on a specific broker.
package messaging

import (
	"context"
	"encoding/json"
)

// Message is the payload handed to subscription handlers after envelope unwrapping.
type Message struct {
	Version string
	Payload json.RawMessage
}

// MessageHandler handles a single message for a subject.
type MessageHandler func(ctx context.Context, m Message) error

// MessageHandlerRegisterer registers subject handlers; implemented by Router.
type MessageHandlerRegisterer interface {
	Register(subject string, h MessageHandler)
}
