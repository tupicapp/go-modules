package clock

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(fx.Annotate(NewSystem, fx.As(new(Clock)))),
)
