// Package logger provides the Logger contract plus zap, memory, and noop implementations.
package logger

import (
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

// Logger is the interface for logging messages with different levels and structured fields.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// Config selects and configures the logger driver.
type Config struct {
	Driver string // zap | memory | noop
	Level  string // debug | info | warn | error | fatal | panic
	Format string // time layout for log timestamps
	Path   string // output path for the zap driver
	Debug  bool
}

// New builds a Logger for the configured driver.
func New(cfg Config) (Logger, error) {
	switch cfg.Driver {
	case "zap":
		return NewZap(cfg)
	case "memory":
		return NewMemory(), nil
	case "noop":
		return NewNoop(), nil
	default:
		return nil, errors.Newf("logger: unknown driver: %s", cfg.Driver)
	}
}
