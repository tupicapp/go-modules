package authorization

import "context"

// userKey is the context key for the request's resolved user entity. It is
// parameterized by the user type so each service's user is stored under a
// distinct, collision-free key.
type userKey[U any] struct{}

// ContextWithUser returns a new context carrying the service's user entity.
// The auth middleware sets it after authentication; handlers read it with
// UserFromContext.
func ContextWithUser[U any](ctx context.Context, u *U) context.Context {
	return context.WithValue(ctx, userKey[U]{}, u)
}

// UserFromContext returns the service's user entity from the context, or nil
// if none is present (e.g. service-account requests, or unauthenticated ones).
func UserFromContext[U any](ctx context.Context) *U {
	u, _ := ctx.Value(userKey[U]{}).(*U)
	return u
}
