package nats

import (
	"context"

	"github.com/tupicapp/go-modules/logger"
	"go.uber.org/zap"
)

// Router maps NATS subjects to MessageHandlers.
type Router struct {
	logger logger.Logger
	routes map[string]MessageHandler
}

func NewRouter(l logger.Logger) *Router {
	return &Router{
		logger: l,
		routes: make(map[string]MessageHandler),
	}
}

func (r *Router) Register(subject string, h MessageHandler) {
	if _, exists := r.routes[subject]; exists {
		panic("nats: duplicate handler registration for subject " + subject)
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

func (r *Router) Handle(ctx context.Context, subject string, m Message) error {
	h, ok := r.routes[subject]
	if !ok {
		r.logger.Warn("nats: unhandled subject, skipping", zap.String("subject", subject))
		return nil
	}
	return h(ctx, m)
}

var _ MessageHandlerRegisterer = (*Router)(nil)
