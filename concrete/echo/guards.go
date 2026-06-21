package echo

import (
	base "github.com/labstack/echo/v5"
	"github.com/tupicapp/go-modules/shared/authorization"
)

// RequireUser returns 401 if no authenticated actor is in context.
func RequireUser(next base.HandlerFunc) base.HandlerFunc {
	return func(c *base.Context) error {
		if authorization.ActorFromContext(c.Request().Context()) == nil {
			return authorization.ErrAuthenticationRequired
		}
		return next(c)
	}
}

// RequireAdmin returns 401 if unauthenticated, 403 if the actor is not an admin.
func RequireAdmin(next base.HandlerFunc) base.HandlerFunc {
	return func(c *base.Context) error {
		actor := authorization.ActorFromContext(c.Request().Context())
		if actor == nil {
			return authorization.ErrAuthenticationRequired
		}
		if !actor.IsAdmin {
			return authorization.ErrNotAdminActor
		}
		return next(c)
	}
}

// RequireService returns 401 if unauthenticated, 403 if the actor is not a service.
func RequireService(next base.HandlerFunc) base.HandlerFunc {
	return func(c *base.Context) error {
		actor := authorization.ActorFromContext(c.Request().Context())
		if actor == nil {
			return authorization.ErrAuthenticationRequired
		}
		if actor.Type != authorization.ActorTypeService {
			return authorization.ErrNotServiceActor
		}
		return next(c)
	}
}
