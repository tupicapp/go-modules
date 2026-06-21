package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/contract/logger"
	"github.com/tupicapp/go-modules/testkit/loggertest"
	"go.uber.org/fx"
)

type LifecycleSuite struct {
	suite.Suite
	mem *loggertest.Memory
}

func TestLifecycleSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LifecycleSuite))
}

func (s *LifecycleSuite) SetupTest() {
	s.mem = loggertest.NewMemory()
}

// entry returns the first captured log with the given message.
func (s *LifecycleSuite) entry(msg string) (loggertest.Log, bool) {
	for _, e := range s.mem.Entries() {
		if e.Message == msg {
			return e, true
		}
	}
	return loggertest.Log{}, false
}

// -------- onStart --------

func (s *LifecycleSuite) TestOnStart_LogsAppInfo() {
	info := Info{Name: "svc", Version: "1.2.3", Environment: "test", Debug: true}

	s.Require().NoError(onStart(s.mem, info)(context.Background()))

	e, ok := s.entry("app: starting...")
	s.Require().True(ok)
	s.Equal("info", e.Level)

	fields := map[string]string{}
	for _, f := range e.Fields {
		fields[f.Key] = f.String
	}
	s.Equal("svc", fields["name"])
	s.Equal("1.2.3", fields["version"])
	s.Equal("test", fields["environment"])
}

// -------- onStop --------

func (s *LifecycleSuite) TestOnStop_LogsCleanShutdown() {
	s.Require().NoError(onStop(s.mem)(context.Background()))

	_, stopping := s.entry("app: stopping...")
	s.True(stopping)
	_, stopped := s.entry("app: stopped successfully")
	s.True(stopped)
}

func (s *LifecycleSuite) TestOnStop_WarnsOnExpiredContext() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// An expired shutdown context only warns — the hook itself still succeeds.
	s.Require().NoError(onStop(s.mem)(ctx))

	e, ok := s.entry("app: stopping exceeded the timeout")
	s.Require().True(ok)
	s.Equal("warn", e.Level)
	_, stopped := s.entry("app: stopped successfully")
	s.False(stopped)
}

// -------- Lifecycle --------

func (s *LifecycleSuite) TestLifecycle_RegistersStartAndStopHooks() {
	fxApp := fx.New(
		fx.NopLogger,
		fx.Supply(Info{Name: "svc"}),
		fx.Provide(func() logger.Logger { return s.mem }),
		Lifecycle(),
	)

	s.Require().NoError(fxApp.Start(context.Background()))
	s.Require().NoError(fxApp.Stop(context.Background()))

	_, started := s.entry("app: starting...")
	s.True(started)
	_, stopped := s.entry("app: stopped successfully")
	s.True(stopped)
}
