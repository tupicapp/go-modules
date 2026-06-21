// Package logger provides the Logger contract.
package logger

import (
	"go.uber.org/zap"
)

// Logger is the interface for logging messages with different levels and structured fields.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}
