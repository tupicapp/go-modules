// Package authx selects the platform authenticator driver (IAM or dummy),
// generic over the service's user entity.
package authx

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupic/common-go/authorization"
	"github.com/tupic/common-go/authx/dummy"
	"github.com/tupic/common-go/authx/iam"
)

// Authenticator validates a bearer token and returns the resolved actor and
// the service's user entity.
type Authenticator[U any] interface {
	Authenticate(ctx context.Context, token string) (*authorization.Actor, *U, error)
}

// Config selects the authenticator driver and carries the IAM settings.
type Config struct {
	Driver string // iam | dummy
	IAM    iam.Config
}

// New builds the configured authenticator. resolver provisions users from
// validated IAM claims; findUser looks users up by ID for the dummy driver.
func New[U any](
	cfg Config,
	resolver iam.UserResolver[U],
	findUser func(ctx context.Context, id uuid.UUID) (*U, error),
) (Authenticator[U], error) {
	switch cfg.Driver {
	case "iam":
		return iam.New(cfg.IAM, resolver), nil
	case "dummy":
		return dummy.New(findUser), nil
	default:
		return nil, errors.Newf("auth: unknown driver: %s", cfg.Driver)
	}
}
