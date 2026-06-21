// Package sentry initializes the Sentry SDK with a shutdown flush hook.
package sentry

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/getsentry/sentry-go"
)

// Config carries the Sentry client settings supplied by the service.
type Config struct {
	DSN         string
	Environment string
	Release     string
	Debug       bool
}

// Init initializes the Sentry error tracking SDK. Returns nil if DSN is empty, skipping initialization. The service
// owns the shutdown flush by registering Close on its app lifecycle.
func Init(cfg Config) error {
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

	return nil
}

// Close flushes buffered events; the service registers it as a shutdown hook. It is a no-op when Sentry was never
// initialized (empty DSN), so it is always safe to register.
func Close() {
	sentry.Flush(2 * time.Second)
}

// Capture reports err to Sentry on a cloned hub with the given tags. It is a no-op when err is nil or Sentry was not
// initialized (no DSN). Use at transport boundaries to report errors no handler dealt with, mirroring the HTTP edge.
func Capture(err error, tags map[string]string) {
	if err == nil {
		return
	}

	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(scope *sentry.Scope) {
		for key, value := range tags {
			scope.SetTag(key, value)
		}
		hub.CaptureException(err)
	})
}
