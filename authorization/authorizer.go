package authorization

import (
	"slices"
	"strings"

	"github.com/tupicapp/go-modules/apperror"
)

// Authorizer handles application layer authorization: it checks whether the given actor holds all the required
// permissions.
type Authorizer interface {
	Authorize(actor *Actor, permissions ...string) error
}

// defaultScopes are always present when a user authenticates via the standard platform login flow. Additional scopes
// such as "offline_access" may also be present but do not affect flow detection.
var defaultScopes = []string{"openid", "profile", "email"}

// TokenAuthorizer authorizes tokens based on their scopes and permissions.
//
// Permissions are fully-qualified, prefixed with the owning service: "assets:assets.write",
// "notifications:preferences.read". A scope or permission entry matches when it equals the permission itself, its
// admin-prefixed form ("admin:<permission>"), or a service-level wildcard ("<service>:*" / "admin:<service>:*").
type TokenAuthorizer struct{}

func NewTokenAuthorizer() *TokenAuthorizer { return &TokenAuthorizer{} }

func (a *TokenAuthorizer) Authorize(actor *Actor, permissions ...string) error {
	if actor == nil {
		return apperror.Authentication("Authentication required.")
	}

	// Service actors are internal machine clients — fully trusted, no scope or permission checks.
	if actor.Type == ActorTypeService {
		return nil
	}

	if isStandardFlow(actor.Scopes) {
		return nil
	}

	// For non-standard tokens (API keys, restricted clients), check that the token carries at least one valid scope form
	// for each permission.
	for _, p := range permissions {
		if !hasValidScope(actor.Scopes, p) {
			return apperror.Authentication("Insufficient token scope.")
		}
	}

	for _, p := range permissions {
		if !hasValidScope(actor.Permissions, p) {
			return apperror.Authorization("Insufficient permissions.")
		}
	}

	return nil
}

// isStandardFlow reports whether the token has the default scopes. Their presence grants full access without requiring
// any additional check.
func isStandardFlow(scopes []string) bool {
	for _, s := range defaultScopes {
		if !slices.Contains(scopes, s) {
			return false
		}
	}
	return true
}

// hasValidScope reports whether list contains the user form, admin form, or a service wildcard form of permission.
func hasValidScope(list []string, permission string) bool {
	service := strings.Split(permission, ":")[0]
	userForm := permission
	adminForm := "admin:" + userForm

	for _, s := range list {
		if s == userForm || s == adminForm || s == service+":*" || s == "admin:"+service+":*" {
			return true
		}
	}
	return false
}

var _ Authorizer = (*TokenAuthorizer)(nil)
