package logger

import (
	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/concrete/zap"
	contract "github.com/tupicapp/go-modules/contract/logger"
	"go.uber.org/fx"
)

// Module provides a Logger from a logger.Config supplied by the service.
var Module = fx.Options(
	fx.Provide(newLogger),
)

func newLogger(lc fx.Lifecycle, cfg Config) (contract.Logger, error) {
	l, err := New(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if z, ok := l.(*zap.Zap); ok {
		lc.Append(fx.Hook{OnStop: z.Stop})
	}
	return l, nil
}
