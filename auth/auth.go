// Package auth provides authentication for Tupic services: turning a bearer
// credential into an identity.
//
// # Authentication vs. authorization
//
// This package answers "who is calling?" (credential → identity). Its sibling
// package authorization answers "may they do this?" (identity → permission
// decision). The two meet in the Actor type: authentication produces an
// *authorization.Actor, authorization policies consume it.
//
// # The flow
//
//	bearer token
//	     │
//	     ▼
//	Authenticator[U].Authenticate            (this package: iam or dummy)
//	     │
//	     ├── *authorization.Actor            security context (ID, type,
//	     │                                   scopes, permissions, admin flag)
//	     └── *U                              the service's own user entity,
//	                                         nil for service accounts
//
// U is the service's user entity type. Authentication is generic over it
// because every service owns its user model; the shared code never needs to
// know its fields.
//
// # Drivers
//
// Two drivers ship with the platform:
//
//   - iam: validates Keycloak JWTs against a JWKS endpoint (production).
//   - dummy: decodes the token as a base64 JSON Actor (tests, local dev).
//
// New selects between them from config. A service that needs a custom driver
// (API keys, another IdP, …) implements the one-method Authenticator
// interface — or wraps a closure in Func — and skips New entirely; nothing
// else in the stack cares where the Actor came from.
package auth

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupic/common-go/auth/dummy"
	"github.com/tupic/common-go/auth/iam"
	"github.com/tupic/common-go/authorization"
)

// Authenticator validates a bearer token and returns the resolved actor and
// the service's user entity. The user is nil for service-account actors.
type Authenticator[U any] interface {
	Authenticate(ctx context.Context, token string) (*authorization.Actor, *U, error)
}

// Func adapts a plain function to the Authenticator interface, for custom
// drivers that don't need any state.
type Func[U any] func(ctx context.Context, token string) (*authorization.Actor, *U, error)

func (f Func[U]) Authenticate(ctx context.Context, token string) (*authorization.Actor, *U, error) {
	return f(ctx, token)
}

// Driver names accepted by Config.Driver.
const (
	DriverIAM   = "iam"
	DriverDummy = "dummy"
)

// Config selects the authenticator driver and carries the IAM settings.
type Config struct {
	Driver string // iam | dummy
	IAM    iam.Config
}

// New builds the configured built-in authenticator.
//
// resolver provisions the service's user from validated IAM claims (only used
// by the iam driver); findUser looks users up by ID (only used by the dummy
// driver). Options are forwarded to the iam driver.
func New[U any](
	cfg Config,
	resolver iam.UserResolver[U],
	findUser func(ctx context.Context, id uuid.UUID) (*U, error),
	opts ...iam.Option,
) (Authenticator[U], error) {
	switch cfg.Driver {
	case DriverIAM:
		return iam.New(cfg.IAM, resolver, opts...), nil
	case DriverDummy:
		return dummy.New(findUser), nil
	default:
		return nil, errors.Newf("auth: unknown driver: %s", cfg.Driver)
	}
}
