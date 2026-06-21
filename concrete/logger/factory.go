package logger

import (
	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/concrete/noop_logger"
	"github.com/tupicapp/go-modules/concrete/zap"
	contract "github.com/tupicapp/go-modules/contract/logger"
)

// Config selects the logger driver and nests each driver's own config. Only the selected driver's sub-config is read;
// the factory never restates a driver's fields. The mapstructure/validate tags let a service load it directly (alias it).
type Config struct {
	Driver string     `mapstructure:"driver" validate:"required,oneof=zap noop"`
	Zap    zap.Config `mapstructure:"zap"`
}

// New builds a Logger for the configured driver, constructing only the chosen one.
func New(cfg Config) (contract.Logger, error) {
	switch cfg.Driver {
	case "zap":
		return zap.NewZap(cfg.Zap)
	case "noop":
		return noop_logger.NewNoop(), nil
	default:
		return nil, errors.Newf("logger: unknown driver: %s", cfg.Driver)
	}
}
