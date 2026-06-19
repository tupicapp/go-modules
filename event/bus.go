package event

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/logger"
	"go.uber.org/zap"
)

// Middleware wraps a Handler to add cross-cutting behaviour (logging, metrics,
// tracing, recovery). Middleware passed to NewBus runs outermost-first around
// every handler; a built-in recovery layer always sits closest to the handler
// so a handler panic surfaces as an error to the middleware above it.
type Middleware func(next Handler) Handler

// Bus is an in-memory synchronous event bus for domain events with subscription and publishing capabilities.
type Bus struct {
	logger logger.Logger
	mu     sync.RWMutex
	routes map[string][]Handler
	mw     []Middleware
}

// NewBus builds the bus. Optional middleware wraps every handler; handler panics
// are always recovered and converted to errors regardless of the middleware set.
func NewBus(l logger.Logger, mw ...Middleware) (Publisher, Subscriber) {
	eb := &Bus{
		logger: l,
		routes: make(map[string][]Handler),
		mw:     mw,
	}
	return eb, eb
}

func (eb *Bus) Subscribe(eventName string, h Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.routes[eventName] = append(eb.routes[eventName], eb.wrap(h))
}

// wrap applies recovery closest to the handler, then the configured middleware
// outermost-first, so the chain at dispatch is mw[0] → … → recover → handler.
func (eb *Bus) wrap(h Handler) Handler {
	h = recoverMiddleware(eb.logger)(h)
	for i := len(eb.mw) - 1; i >= 0; i-- {
		h = eb.mw[i](h)
	}
	return h
}

func (eb *Bus) Publish(ctx context.Context, e DomainEvent) error {
	name := e.Name()

	eb.mu.RLock()
	handlers := append([]Handler(nil), eb.routes[name]...)
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		eb.logger.Debug("eventbus: no handlers for domain event", zap.String("event", name))
		return nil
	}

	for i, h := range handlers {
		if err := h(ctx, e); err != nil {
			eb.logger.Warn("eventbus: handler failed; aborting publish",
				zap.String("event", name),
				zap.Int("handler_index", i),
				zap.Error(err),
			)
			return errors.Wrapf(err, "handler %d for %q failed", i, name)
		}
	}

	return nil
}

// PublishAll dispatches events in order, aborting on the first handler error.
func (eb *Bus) PublishAll(ctx context.Context, events []DomainEvent) error {
	for _, e := range events {
		if err := eb.Publish(ctx, e); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
