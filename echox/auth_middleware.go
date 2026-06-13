package echox

import (
	"context"
	"strings"

	labecho "github.com/labstack/echo/v5"
	"github.com/tupic/common-go/authorization"
	authrequest "github.com/tupic/common-go/authorization/requestcontext"
)

// AuthMiddleware resolves the actor and user from the Bearer token; the
// request continues without an actor if the token is absent or invalid.
//
// The middleware is generic over the service's user entity: authenticate is
// the service's authenticator (typically iam or dummy from authx), withUser
// stores the resolved user in the request context using the service's own
// context key, and adminPathPrefix marks the route subtree that requires
// admin-role hydration from the userinfo endpoint (empty disables hydration).
func AuthMiddleware[U any](
	authenticate func(ctx context.Context, token string) (*authorization.Actor, *U, error),
	withUser func(ctx context.Context, u *U) context.Context,
	adminPathPrefix string,
) labecho.MiddlewareFunc {
	return func(next labecho.HandlerFunc) labecho.HandlerFunc {
		return func(c *labecho.Context) error {
			token := BearerToken(c)
			if token == "" {
				return next(c)
			}

			ctx := c.Request().Context()
			if requiresAdminRoleHydration(c.Request().URL.Path, adminPathPrefix) {
				ctx = authrequest.ContextWithAdminRoleHydration(ctx)
			}

			actor, u, err := authenticate(ctx, token)
			if err != nil {
				return next(c)
			}

			ctx = authorization.ContextWithActor(ctx, actor)
			// Service-account tokens authenticate without a user — guard
			// against storing a nil user in the request context.
			if u != nil {
				ctx = withUser(ctx, u)
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
