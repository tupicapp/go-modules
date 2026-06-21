package authorization

import authz "github.com/tupicapp/go-modules/kernel/authorization"

// Authorizer handles application layer authorization.
// It checks whether the given actor holds all the required permissions.
type Authorizer interface {
	Authorize(actor *authz.Actor, permissions ...string) error
}
