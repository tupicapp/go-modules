package random

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(fx.Annotate(NewSecure, fx.As(new(Random)))),
)
