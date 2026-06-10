package validator

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(fx.Annotate(NewPlayground, fx.As(new(Validator)))),
)
