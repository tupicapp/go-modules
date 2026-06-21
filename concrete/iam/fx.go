package iam

import (
	"github.com/tupicapp/go-modules/contract/authentication"
	"go.uber.org/fx"
)

// Module wires the IAM (Keycloak) Authenticator for the service's user entity U. A generic
// constructor cannot be handed to fx.Provide directly (Go has no value for an
// uninstantiated generic function), so the service instantiates Module with its
// concrete user type: iam.Module[user.User](). It expects a Config and the service's
// authentication.UserResolver[U] in the container.
//
// It provides the authentication.Authenticator[U] interface (not the concrete
// type) so a test composition can fx.Decorate it with a dummy authenticator.
func Module[U any]() fx.Option {
	return fx.Provide(func(cfg Config, provisioner authentication.UserResolver[U]) authentication.Authenticator[U] {
		return New(cfg, provisioner)
	})
}
