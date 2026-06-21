package token_authorizer

import (
	"slices"
	"strings"

	contract "github.com/tupicapp/go-modules/contract/authorization"
	authz "github.com/tupicapp/go-modules/kernel/authorization"
)

// defaultScopes are always present when a user authenticates via the standard sign-in flow.
// Additional scopes such as "offline_access" may also be present but do not affect flow detection.
var defaultScopes = []string{"openid", "profile", "email"}

// TokenAuthorizer authorizes tokens based on their scopes and permissions.
//
// Permissions are fully-qualified, prefixed with the owning service ("<service>:<permission>").
// Permission/scope examples:
//   - "assets:assets.write"
//   - "notifications:preferences.read"
//
// Admin permission/scope examples:
//   - "admin:assets:assets.write"
//   - "admin:notifications:preferences.read"
//
// A scope or permission entry matches when it equals the permission itself, its admin-prefixed form
// ("admin:<permission>"), or a service-level wildcard ("<service>:*" / "admin:<service>:*").
type TokenAuthorizer struct{}

func New() *TokenAuthorizer { return &TokenAuthorizer{} }

func (a *TokenAuthorizer) Authorize(actor *authz.Actor, permissions ...string) error {
	if actor == nil {
		return authz.ErrAuthenticationRequired
	}

	// Service actors are internal machine clients — currently, fully trusted, and no scope or permission checks.
	if actor.Type == authz.ActorTypeService {
		return nil
	}

	// Users signed in with standard flow have all scopes by default.
	// Currently, end users have no permission, therefore no permission check is needed.
	if isForStandardFlow(actor.Scopes) {
		return nil
	}

	// For non-standard sign-in flow (API keys, custom clients), check that the token carries all required scopes.
	for _, p := range permissions {
		if !hasValidScope(actor.Scopes, p) {
			return authz.ErrInsufficientTokenScope
		}
	}

	// For non-standard sign-in flow (API keys, custom clients), check that the token carries all required permissions.
	for _, p := range permissions {
		if !hasValidScope(actor.Permissions, p) {
			return authz.ErrInsufficientPermissions
		}
	}

	return nil
}

// isForStandardFlow reports whether the token has the default scopes. Their presence grants full access without requiring
// any additional check.
func isForStandardFlow(scopes []string) bool {
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

var _ contract.Authorizer = (*TokenAuthorizer)(nil)
