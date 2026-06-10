package authorization

import "go.uber.org/fx"

// Module provides the TokenAuthorizer as the Authorizer contract.
var Module = fx.Options(
	fx.Provide(fx.Annotate(NewTokenAuthorizer, fx.As(new(Authorizer)))),
)
