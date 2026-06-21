package noop_logger

import (
	"github.com/tupicapp/go-modules/contract/logger"
	"go.uber.org/zap"
)

type Noop struct{}

func NewNoop() Noop { return Noop{} }

func (Noop) Debug(_ string, _ ...zap.Field) {}
func (Noop) Info(_ string, _ ...zap.Field)  {}
func (Noop) Warn(_ string, _ ...zap.Field)  {}
func (Noop) Error(_ string, _ ...zap.Field) {}

var _ logger.Logger = Noop{}
