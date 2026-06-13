package sentry_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/sentry"
	"go.uber.org/fx"
)

// recordingLifecycle captures hooks appended during Init.
type recordingLifecycle struct{ hooks []fx.Hook }

func (l *recordingLifecycle) Append(h fx.Hook) { l.hooks = append(l.hooks, h) }

type SentrySuite struct{ suite.Suite }

func TestSentrySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SentrySuite))
}

func (s *SentrySuite) TestInit_EmptyDSN_SkipsInitialization() {
	lc := &recordingLifecycle{}

	s.Require().NoError(sentry.Init(lc, sentry.Config{}))
	// With no DSN, initialization is skipped and no flush hook is registered.
	s.Empty(lc.hooks)
}

func (s *SentrySuite) TestInit_InvalidDSN_ReturnsError() {
	lc := &recordingLifecycle{}

	err := sentry.Init(lc, sentry.Config{DSN: "://not-a-valid-dsn"})
	s.Require().Error(err)
	s.Empty(lc.hooks)
}

func (s *SentrySuite) TestInit_ValidDSN_RegistersFlushHook() {
	lc := &recordingLifecycle{}

	err := sentry.Init(lc, sentry.Config{
		DSN:         "https://public@example.com/1",
		Environment: "test",
		Release:     "1.0.0",
	})
	s.Require().NoError(err)
	s.Require().Len(lc.hooks, 1)
	s.NotNil(lc.hooks[0].OnStop)
}
