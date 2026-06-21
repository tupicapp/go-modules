// Package messaging_router provides the default in-memory subject→handler router implementing
// messaging.MessageHandlerRegisterer. It is transport-agnostic: any adapter can register handlers and dispatch
// unwrapped messages through it.
package messaging_router

import (
	"context"

	"github.com/tupicapp/go-modules/contract/logger"
	"github.com/tupicapp/go-modules/contract/messaging"
	"go.uber.org/zap"
)

// Router maps subjects to MessageHandlers.
type Router struct {
	logger logger.Logger
	routes map[string]messaging.MessageHandler
}

func NewRouter(l logger.Logger) *Router {
	return &Router{
		logger: l,
		routes: make(map[string]messaging.MessageHandler),
	}
}

func (r *Router) Register(subject string, h messaging.MessageHandler) {
	if _, exists := r.routes[subject]; exists {
		panic("messaging: duplicate handler registration for subject " + subject)
	}
	r.routes[subject] = h
}

func (r *Router) Subjects() []string {
	subjects := make([]string, 0, len(r.routes))
	for s := range r.routes {
		subjects = append(subjects, s)
	}
	return subjects
}

func (r *Router) Handle(ctx context.Context, subject string, m messaging.Message) error {
	h, ok := r.routes[subject]
	if !ok {
		r.logger.Warn("messaging: unhandled subject, skipping", zap.String("subject", subject))
		return nil
	}
	return h(ctx, m)
}

var _ messaging.MessageHandlerRegisterer = (*Router)(nil)
