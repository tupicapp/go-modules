// Package authenticationtest provides a test-double authenticator that skips real IAM: it decodes the actor from the
// bearer token and resolves the user through a caller-supplied lookup, so the double carries no persistence dependency.
package authenticationtest

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/tupicapp/go-modules/kernel/authorization"
)

// Authenticator decodes the bearer token as a base64-encoded JSON Actor and resolves the user via lookup. The lookup
// owns persistence concerns (querying, not-found mapping), keeping this double free of any storage dependency.
type Authenticator[U any] struct {
	lookup func(ctx context.Context, id uuid.UUID) (*U, error)
}

func New[U any](lookup func(ctx context.Context, id uuid.UUID) (*U, error)) *Authenticator[U] {
	return &Authenticator[U]{lookup: lookup}
}

func (a *Authenticator[U]) Authenticate(ctx context.Context, token string) (*authorization.Actor, *U, error) {
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

	u, err := a.lookup(ctx, actor.ID)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return &actor, u, nil
}

// EnsureRoles is a no-op: the token already encodes the full actor, including its permissions and admin flag.
func (a *Authenticator[U]) EnsureRoles(
	_ context.Context, _ string, actor *authorization.Actor,
) (*authorization.Actor, error) {
	return actor, nil
}
