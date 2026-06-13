// Package sentry initializes the Sentry SDK with a shutdown flush hook.
package sentry

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/getsentry/sentry-go"
	"go.uber.org/fx"
)

// Config carries the Sentry client settings supplied by the service.
type Config struct {
	DSN         string
	Environment string
	Release     string
	Debug       bool
}

// Init initializes the Sentry error tracking SDK and registers a lifecycle hook to flush events on application
// shutdown. Returns nil if DSN is empty, skipping initialization.
func Init(lc fx.Lifecycle, cfg Config) error {
	if cfg.DSN == "" {
		return nil
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.DSN,
		Environment: cfg.Environment,
		Release:     cfg.Release,
		Debug:       cfg.Debug,
	}); err != nil {
		return errors.Wrap(err, "sentry: initialization failed")
	}

	lc.Append(fx.Hook{
		OnStop: func(c context.Context) error {
			sentry.Flush(2 * time.Second)
			return nil
		},
	})

	return nil
}

// Module invokes Init; requires a sentry.Config in the graph.
var Module = fx.Options(
	fx.Invoke(Init),
)
