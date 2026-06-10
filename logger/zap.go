package logger

import (
	"context"
	"syscall"

	"github.com/cockroachdb/errors"
	base "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Zap struct {
	logger *base.Logger
}

func NewZap(c Config) (*Zap, error) {
	level := base.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(c.Level)); err != nil {
		return nil, errors.Wrapf(err, "logger: invalid level %q", c.Level)
	}

	z, err := base.Config{
		Level:             level,
		Development:       c.Debug,
		Encoding:          "json",
		DisableStacktrace: false,
		DisableCaller:     false,
		OutputPaths:       []string{c.Path},
		ErrorOutputPaths:  []string{c.Path},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			EncodeTime:     zapcore.TimeEncoderOfLayout(c.Format),
			EncodeDuration: zapcore.StringDurationEncoder,
			LevelKey:       "level",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			NameKey:        "key",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			LineEnding:     zapcore.DefaultLineEnding,
		},
	}.Build()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Zap{logger: z}, nil
}

func (z *Zap) Debug(msg string, fields ...base.Field) { z.logger.Debug(msg, fields...) }
func (z *Zap) Info(msg string, fields ...base.Field)  { z.logger.Info(msg, fields...) }
func (z *Zap) Warn(msg string, fields ...base.Field)  { z.logger.Warn(msg, fields...) }
func (z *Zap) Error(msg string, fields ...base.Field) { z.logger.Error(msg, fields...) }

func (z *Zap) Stop(_ context.Context) error {
	if err := z.logger.Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) {
		return errors.WithStack(err)
	}
	return nil
}

var _ Logger = (*Zap)(nil)
