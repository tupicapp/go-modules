package app

import (
	"context"

	"github.com/tupicapp/go-modules/contract/logger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Lifecycle returns a fx application lifecycle including application start and stop hooks.
func Lifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, l logger.Logger, info Info) {
		lc.Append(fx.Hook{
			OnStart: onStart(l, info),
			OnStop:  onStop(l),
		})
	})
}

func onStart(l logger.Logger, info Info) func(context.Context) error {
	return func(ctx context.Context) error {
		l.Info("app: starting...",
			zap.String("name", info.Name),
			zap.String("version", info.Version),
			zap.String("environment", info.Environment),
			zap.Bool("debug", info.Debug),
		)
		return nil
	}
}

func onStop(l logger.Logger) func(context.Context) error {
	return func(ctx context.Context) error {
		l.Info("app: stopping...")
		if err := ctx.Err(); err != nil {
			l.Warn("app: stopping exceeded the timeout", zap.Error(err))
			return nil
		}
		l.Info("app: stopped successfully")
		return nil
	}
}
