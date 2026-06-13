// Package dummy provides an Authenticator that decodes the bearer token as a base64-encoded JSON Actor. Useful in tests
// where no real IAM server is available.
package dummy

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupicapp/common-go/authorization"
)

// Authenticator decodes the bearer token as a base64-encoded JSON Actor. For user actors it fetches the full user
// record via the service-supplied findUser function so callers receive a populated user entity (matching the behavior
// of the IAM authenticator).
type Authenticator[U any] struct {
	findUser func(ctx context.Context, id uuid.UUID) (*U, error)
}

func New[U any](findUser func(ctx context.Context, id uuid.UUID) (*U, error)) *Authenticator[U] {
	return &Authenticator[U]{findUser: findUser}
}

func (a *Authenticator[U]) Authenticate(
	ctx context.Context, token string,
) (*authorization.Actor, *U, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	var actor authorization.Actor
	if err = json.Unmarshal(data, &actor); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if actor.Type != authorization.ActorTypeUser {
		return &actor, nil, nil
	}

	u, err := a.findUser(ctx, actor.ID)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return &actor, u, nil
}

// EnsureRoles is a no-op: the dummy token already encodes the full actor, including its permissions and admin flag.
func (a *Authenticator[U]) EnsureRoles(
	_ context.Context, _ string, actor *authorization.Actor,
) (*authorization.Actor, error) {
	return actor, nil
}
