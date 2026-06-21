package echo

import (
	"go.uber.org/fx"
)

// Module wires the Echo HTTP stack for the service's user entity U. NewEcho is generic over U (it
// consumes authentication.Authenticator[U]) and a generic constructor cannot be handed to fx.Provide
// directly, so the service instantiates Module with its concrete user type: echo.Module[user.User]().
func Module[U any]() fx.Option {
	return fx.Options(
		fx.Provide(
			NewEcho[U],
			NewServer,
		),
		fx.Invoke(RegisterServerLifecycle),
	)
}

func RegisterServerLifecycle(lc fx.Lifecycle, s *Server) {
	lc.Append(fx.Hook{OnStart: s.Start, OnStop: s.Stop})
}
