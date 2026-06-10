package event

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/tupic/common-go/logger"
	"go.uber.org/zap"
)

// Bus is an in-memory synchronous event bus for domain events with
// subscription and publishing capabilities.
type Bus struct {
	logger logger.Logger
	mu     sync.RWMutex
	routes map[string][]Handler
}

func NewBus(l logger.Logger) (Publisher, Subscriber) {
	eb := &Bus{
		logger: l,
		routes: make(map[string][]Handler),
	}
	return eb, eb
}

func (eb *Bus) Subscribe(eventName string, h Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.routes[eventName] = append(eb.routes[eventName], h)
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
