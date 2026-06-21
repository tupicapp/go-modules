// Package objectstorage_router provides the default in-memory event router implementing
// objectstorage.EventHandlerRegisterer; "<Category>:*" matches every subtype within the category, with exact matches
// taking precedence. It is transport-agnostic — any adapter dispatches parsed events through it.
package objectstorage_router

import (
	"strings"

	"github.com/tupicapp/go-modules/contract/objectstorage"
)

// Router maps object-storage event names to EventHandlers.
type Router struct {
	routes map[string]objectstorage.EventHandler
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]objectstorage.EventHandler)}
}

func (r *Router) Register(eventName string, h objectstorage.EventHandler) {
	r.routes[eventName] = h
}

func (r *Router) Handle(e objectstorage.Event) error {
	if h, ok := r.routes[e.EventName]; ok {
		return h(e)
	}
	for pattern, h := range r.routes {
		if matchesPattern(pattern, e.EventName) {
			return h(e)
		}
	}
	return nil
}

func matchesPattern(pattern, eventName string) bool {
	if !strings.HasSuffix(pattern, ":*") {
		return false
	}
	prefix := strings.TrimSuffix(pattern, ":*")
	return strings.HasPrefix(eventName, prefix+":")
}

var _ objectstorage.EventHandlerRegisterer = (*Router)(nil)
