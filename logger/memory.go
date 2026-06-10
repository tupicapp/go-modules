package logger

import (
	"sync"

	"go.uber.org/zap"
)

type Log struct {
	Level   string
	Message string
	Fields  []zap.Field
}

type Memory struct {
	mu      sync.Mutex
	entries []Log
}

func NewMemory() *Memory { return &Memory{} }

func (m *Memory) Entries() []Log {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]Log, 0, len(m.entries))
	return append(cp, m.entries...)
}

func (m *Memory) append(level, msg string, fields []zap.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, Log{Level: level, Message: msg, Fields: fields})
}

func (m *Memory) Debug(msg string, fields ...zap.Field) { m.append("debug", msg, fields) }
func (m *Memory) Info(msg string, fields ...zap.Field)  { m.append("info", msg, fields) }
func (m *Memory) Warn(msg string, fields ...zap.Field)  { m.append("warn", msg, fields) }
func (m *Memory) Error(msg string, fields ...zap.Field) { m.append("error", msg, fields) }

var _ Logger = (*Memory)(nil)
