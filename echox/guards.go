package echox

import (
	labecho "github.com/labstack/echo/v5"
	"github.com/tupic/common-go/apperror"
	"github.com/tupic/common-go/authorization"
)

// RequireUser returns 401 if no authenticated actor is in context.
func RequireUser(next labecho.HandlerFunc) labecho.HandlerFunc {
	return func(c *labecho.Context) error {
		if authorization.ActorFromContext(c.Request().Context()) == nil {
			return apperror.Authentication("Authentication required.")
		}
		return next(c)
	}
}

// RequireAdmin returns 401 if unauthenticated, 403 if the actor is not an
// admin.
func RequireAdmin(next labecho.HandlerFunc) labecho.HandlerFunc {
	return func(c *labecho.Context) error {
		actor := authorization.ActorFromContext(c.Request().Context())
		if actor == nil {
			return apperror.Authentication("Authentication required.")
		}
		if !actor.IsAdmin {
			return apperror.Authorization("Admin access required.")
		}
		return next(c)
	}
}

// RequireService returns 401 if unauthenticated, 403 if the actor is not a
// service.
func RequireService(next labecho.HandlerFunc) labecho.HandlerFunc {
	return func(c *labecho.Context) error {
		actor := authorization.ActorFromContext(c.Request().Context())
		if actor == nil {
			return apperror.Authentication("Authentication required.")
		}
		if actor.Type != authorization.ActorTypeService {
			return apperror.Authorization("Service credentials required.")
		}
		return next(c)
	}
}
