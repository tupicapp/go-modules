package clocktest_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/testkit/clocktest"
)

type FixedSuite struct{ suite.Suite }

func TestFixedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(FixedSuite))
}

func (s *FixedSuite) TestFixed_Now_ReturnsConfiguredTime() {
	t := time.Date(2026, time.June, 13, 10, 30, 0, 0, time.UTC)
	c := clocktest.NewFixed(t)

	s.Equal(t, c.Now())
}

func (s *FixedSuite) TestFixed_Now_IsStableAcrossCalls() {
	c := clocktest.NewFixed(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))

	s.Equal(c.Now(), c.Now())
}
