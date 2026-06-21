package system_clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/system_clock"
)

type ClockSuite struct{ suite.Suite }

func TestClockSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ClockSuite))
}

func (s *ClockSuite) TestSystem_Now_ReturnsCurrentTime() {
	before := time.Now()
	got := system_clock.NewSystem().Now()
	after := time.Now()

	// System.Now must fall within the window the call was made in.
	s.False(got.Before(before), "Now() must not predate the call")
	s.False(got.After(after), "Now() must not postdate the call")
}
