package sentry_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/sentry"
)

type SentrySuite struct{ suite.Suite }

func TestSentrySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SentrySuite))
}

func (s *SentrySuite) TestInit_EmptyDSN_SkipsInitialization() {
	// With no DSN, initialization is skipped and returns no error.
	s.Require().NoError(sentry.Init(sentry.Config{}))
}

func (s *SentrySuite) TestInit_InvalidDSN_ReturnsError() {
	err := sentry.Init(sentry.Config{DSN: "://not-a-valid-dsn"})
	s.Require().Error(err)
}

func (s *SentrySuite) TestInit_ValidDSN_Succeeds() {
	err := sentry.Init(sentry.Config{
		DSN:         "https://public@example.com/1",
		Environment: "test",
		Release:     "1.0.0",
	})
	s.Require().NoError(err)
}

func (s *SentrySuite) TestClose_IsSafeWithoutInitialization() {
	// Close must be safe to call even when Sentry was never initialized.
	s.Require().NotPanics(sentry.Close)
}
