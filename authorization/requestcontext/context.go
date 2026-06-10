// Package requestcontext carries per-request authentication hints between the
// HTTP layer and authenticators.
package requestcontext

import "context"

type hydrateAdminRolesKey struct{}

// ContextWithAdminRoleHydration marks a request context so authenticators know
// admin roles may be required for the current request.
func ContextWithAdminRoleHydration(ctx context.Context) context.Context {
	return context.WithValue(ctx, hydrateAdminRolesKey{}, true)
}

// ShouldHydrateAdminRoles reports whether the request context requires admin
// role hydration beyond what is already present in the token.
func ShouldHydrateAdminRoles(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	value, ok := ctx.Value(hydrateAdminRolesKey{}).(bool)
	return ok && value
}
