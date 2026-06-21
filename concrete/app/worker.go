package app

import (
	"github.com/tupicapp/go-modules/concrete/worker"
	"go.uber.org/fx"
)

// NewWorkerApp creates a worker application from the given modules, activating
// only the named subscriptions. An empty subscriptions slice activates all subscriptions.
func NewWorkerApp(modules fx.Option, subscriptions []string) *fx.App {
	return fx.New(modules, fx.Supply(worker.Filter{Subscriptions: subscriptions}))
}
