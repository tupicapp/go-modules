package echo

import (
	"context"

	labecho "github.com/labstack/echo/v5"
	"github.com/tupicapp/go-modules/authorization"
)

// AuthConfig wires the shared auth middleware to a service. New fields can be added without breaking existing call
// sites — prefer extending this struct over adding parameters.
type AuthConfig[U any] struct {
	// Authenticate validates the bearer token and returns the actor and the service's user entity (typically the auth
	// package's Authenticate).
	Authenticate func(ctx context.Context, token string) (*authorization.Actor, *U, error)
}

// AuthMiddleware resolves the actor and user from the Bearer token; the request continues without an actor if the token
// is absent or invalid — route guards (RequireUser, RequireAdmin, RequireService) decide whether an anonymous request
// may proceed.
//
// The actor's realm roles are taken from the token as-is. Routes that need them complete (admin routes) add EnsureRoles
// to their group.
func AuthMiddleware[U any](cfg AuthConfig[U]) labecho.MiddlewareFunc {
	return func(next labecho.HandlerFunc) labecho.HandlerFunc {
		return func(c *labecho.Context) error {
			token := BearerToken(c)
			if token == "" {
				return next(c)
			}

			actor, u, err := cfg.Authenticate(c.Request().Context(), token)
			if err != nil {
				return next(c)
			}

			ctx := authorization.ContextWithActor(c.Request().Context(), actor)
			// Service-account tokens authenticate without a user — guard against storing a nil user in the request context.
			if u != nil {
				ctx = authorization.ContextWithUser(ctx, u)
			}
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// EnsureRoles returns middleware that completes the authenticated actor's realm roles before the route runs, fetching
// them from the identity provider when the access token omitted them. Apply it to the admin route group, ahead of
// RequireAdmin — it is the explicit declaration that this subtree needs accurate roles.
//
// ensure is typically the auth driver's EnsureRoles method. Failures fail closed: the un-hydrated actor proceeds, so
// RequireAdmin denies access rather than letting a userinfo outage 500 the request.
func EnsureRoles(
	ensure func(ctx context.Context, token string, actor *authorization.Actor) (*authorization.Actor, error),
) labecho.MiddlewareFunc {
	return func(next labecho.HandlerFunc) labecho.HandlerFunc {
		return func(c *labecho.Context) error {
			actor := authorization.ActorFromContext(c.Request().Context())
			if actor == nil {
				return next(c)
			}

			hydrated, err := ensure(c.Request().Context(), BearerToken(c), actor)
			if err != nil {
				// Fail closed: proceed with the un-hydrated actor; RequireAdmin will deny since its roles could not be confirmed.
				return next(c)
			}

			ctx := authorization.ContextWithActor(c.Request().Context(), hydrated)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
