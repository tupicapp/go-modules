package echo

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	base "github.com/labstack/echo/v5"
)

const sentryHubKey = "sentry_hub"

// Sentry returns a middleware that attaches a per-request Sentry hub to the echo context and captures any panics before
// the Recover middleware handles them.
func Sentry() base.MiddlewareFunc {
	return func(next base.HandlerFunc) base.HandlerFunc {
		return func(c *base.Context) (err error) {
			hub := sentry.CurrentHub().Clone()
			hub.Scope().SetRequest(c.Request())
			c.Set(sentryHubKey, hub)
			defer func() {
				if r := recover(); r != nil {
					hub.Recover(r)
					panic(r)
				}
			}()
			return next(c)
		}
	}
}

// GetSentryHub returns the per-request Sentry hub stored in the echo context, or nil when Sentry is not configured.
func GetSentryHub(c *base.Context) *sentry.Hub {
	v := c.Get(sentryHubKey)
	if v == nil {
		return nil
	}
	hub, ok := v.(*sentry.Hub)
	if !ok {
		panic(fmt.Sprintf("middleware: sentryHubKey holds %T, want *sentry.Hub", v))
	}
	return hub
}
