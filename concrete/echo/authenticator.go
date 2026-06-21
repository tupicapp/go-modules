package echo

import (
	"strings"

	base "github.com/labstack/echo/v5"
	"github.com/tupicapp/go-modules/contract/authentication"
	"github.com/tupicapp/go-modules/shared/authorization"
)

func Authenticator[U any](a authentication.Authenticator[U]) base.MiddlewareFunc {
	return func(next base.HandlerFunc) base.HandlerFunc {
		return func(c *base.Context) error {
			token := BearerToken(c)
			if token == "" {
				return next(c)
			}

			actor, u, err := a.Authenticate(c.Request().Context(), token)
			if err != nil {
				return next(c)
			}

			if actor.IsAdmin {
				actorWithRoles, err := a.EnsureRoles(c.Request().Context(), BearerToken(c), actor)
				if err == nil {
					actor = actorWithRoles
				}
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

// BearerToken extracts the Bearer token from the Authorization header, returning an empty string if not present.
func BearerToken(c *base.Context) string {
	h := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
