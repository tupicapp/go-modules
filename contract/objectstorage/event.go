// Package objectstorage defines the transport-agnostic object-storage event
// contract shared by storage-event adapters (SQS, SNS, EventBridge, …). An
// adapter parses its source notification into an Event and dispatches it through
// a Router; application handlers depend only on this package, never on the
// delivery mechanism.
package objectstorage

// Event is a parsed, URL-decoded object-storage record (e.g. an S3 object
// notification) extracted from a delivery message.
type Event struct {
	EventName string
	Bucket    string
	Key       string
	Size      *int64
}

// EventHandler processes a single object-storage event.
type EventHandler func(e Event) error

// EventHandlerRegisterer maps event names (e.g. "ObjectCreated:*") to handlers;
// implemented by Router.
type EventHandlerRegisterer interface {
	Register(eventName string, h EventHandler)
}
