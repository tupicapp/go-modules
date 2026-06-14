package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/clock"
)

type ClockSuite struct{ suite.Suite }

func TestClockSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ClockSuite))
}

func (s *ClockSuite) TestSystem_Now_ReturnsCurrentTime() {
	before := time.Now()
	got := clock.NewSystem().Now()
	after := time.Now()

	// System.Now must fall within the window the call was made in.
	s.False(got.Before(before), "Now() must not predate the call")
	s.False(got.After(after), "Now() must not postdate the call")
}

func (s *ClockSuite) TestFixed_Now_ReturnsConfiguredTime() {
	t := time.Date(2026, time.June, 13, 10, 30, 0, 0, time.UTC)
	c := clock.NewFixed(t)

	s.Equal(t, c.Now())
}

func (s *ClockSuite) TestFixed_Now_IsStableAcrossCalls() {
	c := clock.NewFixed(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))

	s.Equal(c.Now(), c.Now())
}
