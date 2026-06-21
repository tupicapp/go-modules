package authentication

import (
	"context"

	"github.com/tupicapp/go-modules/shared/authorization"
)

// Authenticator validates a bearer token and returns the resolved actor and the service's user entity. The user is nil
// for service-account actors.
//
// EnsureRoles completes the actor's realm roles for routes that need them (admin routes), fetching from the identity
// provider only when the token omitted roles. It is a no-op when roles are already present or when the driver embeds
// them in the credential, so it is safe to call on any actor.
type Authenticator[U any] interface {
	Authenticate(ctx context.Context, token string) (*authorization.Actor, *U, error)
	EnsureRoles(ctx context.Context, token string, actor *authorization.Actor) (*authorization.Actor, error)
}
