package echox

import (
	"context"
	"strings"

	labecho "github.com/labstack/echo/v5"
	"github.com/tupic/common-go/authorization"
	authrequest "github.com/tupic/common-go/authorization/requestcontext"
)

// AuthConfig wires the shared auth middleware to a service. New fields can be
// added without breaking existing call sites — prefer extending this struct
// over adding parameters.
type AuthConfig[U any] struct {
	// Authenticate validates the bearer token and returns the actor and the
	// service's user entity (typically the auth package's Authenticate).
	Authenticate func(ctx context.Context, token string) (*authorization.Actor, *U, error)

	// WithUser stores the resolved user in the request context using the
	// service's own context key, so downstream handlers can fetch it with
	// the service's UserFromContext.
	WithUser func(ctx context.Context, u *U) context.Context

	// AdminPathPrefix marks the route subtree whose requests need admin-role
	// hydration from the userinfo endpoint (e.g. "/assets/v1/admin").
	// Empty disables hydration.
	AdminPathPrefix string
}

// AuthMiddleware resolves the actor and user from the Bearer token; the
// request continues without an actor if the token is absent or invalid —
// route guards (RequireUser, RequireAdmin, RequireService) decide whether an
// anonymous request may proceed.
func AuthMiddleware[U any](cfg AuthConfig[U]) labecho.MiddlewareFunc {
	return func(next labecho.HandlerFunc) labecho.HandlerFunc {
		return func(c *labecho.Context) error {
			token := BearerToken(c)
			if token == "" {
				return next(c)
			}

			ctx := c.Request().Context()
			if requiresAdminRoleHydration(c.Request().URL.Path, cfg.AdminPathPrefix) {
				ctx = authrequest.ContextWithAdminRoleHydration(ctx)
			}

			actor, u, err := cfg.Authenticate(ctx, token)
			if err != nil {
				return next(c)
			}

			ctx = authorization.ContextWithActor(ctx, actor)
			// Service-account tokens authenticate without a user — guard
			// against storing a nil user in the request context.
			if u != nil {
				ctx = cfg.WithUser(ctx, u)
			}
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func requiresAdminRoleHydration(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}
