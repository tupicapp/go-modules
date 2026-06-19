package event

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/logger"
	"go.uber.org/zap"
)

// recoverMiddleware converts a handler panic into an error so a single faulty
// handler cannot crash the publisher (and, for in-transaction domain events,
// triggers a clean rollback instead). It is always applied by the bus, closest
// to the handler, so the error flows back out through any user middleware.
func recoverMiddleware(l logger.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, e DomainEvent) (err error) {
			defer func() {
				if r := recover(); r != nil {
					l.Error("eventbus: handler panicked", zap.String("event", e.Name()), zap.Any("panic", r))
					err = errors.WithStack(fmt.Errorf("handler for %q panicked: %v", e.Name(), r))
				}
			}()
			return next(ctx, e)
		}
	}
}
