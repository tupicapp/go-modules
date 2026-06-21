// Package outboxtest provides recording test doubles for the outbox port.
package outboxtest

import (
	"context"
	"sync"

	"github.com/tupicapp/go-modules/contract/outbox"
)

// Recorder is a thread-safe recording Outbox for tests. It captures every Store
// call so tests can assert which events were emitted without a live message bus.
//
// Usage in a suite test:
//
//	rec.Reset()                                   // optional: start fresh mid-suite
//	// … exercise the use case …
//	events := rec.EventsBySubject("events.asset.created")
//	require.Len(t, events, 1)
type Recorder struct {
	mu     sync.Mutex
	events []outbox.IntegrationEvent
}

// New returns an empty Recorder.
func New() *Recorder { return &Recorder{} }

// Store records the event and always returns nil.
func (r *Recorder) Store(_ context.Context, event outbox.IntegrationEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

// Events returns a snapshot of all events recorded since the last Reset (or creation).
func (r *Recorder) Events() []outbox.IntegrationEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]outbox.IntegrationEvent{}, r.events...)
}

// EventsBySubject returns only the events whose Subject() matches the given string.
func (r *Recorder) EventsBySubject(subject string) []outbox.IntegrationEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []outbox.IntegrationEvent
	for _, e := range r.events {
		if e.Subject() == subject {
			out = append(out, e)
		}
	}
	return out
}

// Reset discards all recorded events. Call it in SetupTest for a clean slate per test method.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = nil
}

var _ outbox.Outbox = (*Recorder)(nil)
