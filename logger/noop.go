package logger

import "go.uber.org/zap"

type Noop struct{}

func NewNoop() Noop { return Noop{} }

func (Noop) Debug(_ string, _ ...zap.Field) {}
func (Noop) Info(_ string, _ ...zap.Field)  {}
func (Noop) Warn(_ string, _ ...zap.Field)  {}
func (Noop) Error(_ string, _ ...zap.Field) {}

var _ Logger = Noop{}
