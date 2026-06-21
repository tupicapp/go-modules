package app

import "go.uber.org/fx"

// NewServerApp creates a server application from the given modules.
func NewServerApp(modules fx.Option) *fx.App {
	return fx.New(modules)
}
