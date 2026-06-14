package logger_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/logger"
	"go.uber.org/zap"
)

type LoggerSuite struct{ suite.Suite }

func TestLoggerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LoggerSuite))
}

// -------- New (driver dispatch) --------

func (s *LoggerSuite) TestNew_MemoryDriver() {
	l, err := logger.New(logger.Config{Driver: "memory"})
	s.Require().NoError(err)
	s.IsType(&logger.Memory{}, l)
}

func (s *LoggerSuite) TestNew_NoopDriver() {
	l, err := logger.New(logger.Config{Driver: "noop"})
	s.Require().NoError(err)
	s.IsType(logger.Noop{}, l)
}

func (s *LoggerSuite) TestNew_ZapDriver() {
	l, err := logger.New(logger.Config{
		Driver: "zap",
		Level:  "info",
		Format: time.RFC3339,
		Path:   "stderr",
	})
	s.Require().NoError(err)
	s.IsType(&logger.Zap{}, l)
}

func (s *LoggerSuite) TestNew_UnknownDriver_ReturnsError() {
	_, err := logger.New(logger.Config{Driver: "bogus"})
	s.Require().Error(err)
	s.ErrorContains(err, "bogus")
}

// -------- Memory --------

func (s *LoggerSuite) TestMemory_CapturesLevelMessageAndFields() {
	m := logger.NewMemory()
	m.Info("hello", zap.String("k", "v"))
	m.Error("boom")

	entries := m.Entries()
	s.Require().Len(entries, 2)

	s.Equal("info", entries[0].Level)
	s.Equal("hello", entries[0].Message)
	s.Require().Len(entries[0].Fields, 1)
	s.Equal("k", entries[0].Fields[0].Key)

	s.Equal("error", entries[1].Level)
	s.Equal("boom", entries[1].Message)
}

func (s *LoggerSuite) TestMemory_EntriesReturnsCopy() {
	m := logger.NewMemory()
	m.Info("first")

	snapshot := m.Entries()
	m.Info("second")

	// The earlier snapshot must not see entries appended afterwards.
	s.Len(snapshot, 1)
	s.Len(m.Entries(), 2)
}

// -------- Noop --------

func (s *LoggerSuite) TestNoop_DoesNotPanic() {
	n := logger.NewNoop()
	s.NotPanics(func() {
		n.Debug("d")
		n.Info("i")
		n.Warn("w")
		n.Error("e", zap.Int("n", 1))
	})
}

// -------- Zap --------

func (s *LoggerSuite) TestZap_InvalidLevel_ReturnsError() {
	_, err := logger.NewZap(logger.Config{Level: "screaming", Path: "stderr"})
	s.Require().Error(err)
}

func (s *LoggerSuite) TestZap_WritesJSONToOutputPath() {
	path := filepath.Join(s.T().TempDir(), "app.log")
	z, err := logger.NewZap(logger.Config{Level: "info", Format: time.RFC3339, Path: path})
	s.Require().NoError(err)

	z.Info("written", zap.String("key", "value"))
	s.Require().NoError(z.Stop(context.Background()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	out := string(data)
	s.Contains(out, `"message":"written"`)
	s.Contains(out, `"key":"value"`)
	s.True(strings.HasPrefix(out, "{"), "zap output should be JSON")
}

func (s *LoggerSuite) TestZap_RespectsLevelThreshold() {
	path := filepath.Join(s.T().TempDir(), "app.log")
	z, err := logger.NewZap(logger.Config{Level: "warn", Format: time.RFC3339, Path: path})
	s.Require().NoError(err)

	z.Info("suppressed")
	z.Warn("kept")
	s.Require().NoError(z.Stop(context.Background()))

	data, err := os.ReadFile(path)
	s.Require().NoError(err)
	out := string(data)
	s.NotContains(out, "suppressed")
	s.Contains(out, "kept")
}
