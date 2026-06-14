package random_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/random"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

type RandomSuite struct{ suite.Suite }

func TestRandomSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RandomSuite))
}

func (s *RandomSuite) TestString_ReturnsRequestedLength() {
	got, err := random.NewSecure().String(32)
	s.Require().NoError(err)
	s.Len(got, 32)
}

func (s *RandomSuite) TestString_UsesOnlyAllowedCharset() {
	got, err := random.NewSecure().String(128)
	s.Require().NoError(err)
	for _, r := range got {
		s.True(strings.ContainsRune(charset, r), "unexpected character %q", r)
	}
}

func (s *RandomSuite) TestString_Zero_ReturnsEmptyWithoutError() {
	got, err := random.NewSecure().String(0)
	s.Require().NoError(err)
	s.Empty(got)
}

func (s *RandomSuite) TestString_Negative_ReturnsError() {
	_, err := random.NewSecure().String(-1)
	s.Require().Error(err)
}

func (s *RandomSuite) TestString_ProducesDistinctValues() {
	r := random.NewSecure()
	a, err := r.String(32)
	s.Require().NoError(err)
	b, err := r.String(32)
	s.Require().NoError(err)
	// A collision at 32 chars over a 64-symbol alphabet is astronomically unlikely.
	s.NotEqual(a, b)
}
