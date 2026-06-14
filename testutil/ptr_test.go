package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/testutil"
)

type PtrSuite struct{ suite.Suite }

func TestPtrSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PtrSuite))
}

func (s *PtrSuite) TestPtr_String_ReturnsPointerToValue() {
	p := testutil.Ptr("hello")
	s.Require().NotNil(p)
	s.Equal("hello", *p)
}

func (s *PtrSuite) TestPtr_Int_ReturnsPointerToValue() {
	p := testutil.Ptr(42)
	s.Require().NotNil(p)
	s.Equal(42, *p)
}

func (s *PtrSuite) TestPtr_DistinctValuesYieldDistinctPointers() {
	a, b := testutil.Ptr(1), testutil.Ptr(2)
	s.NotSame(a, b)
	s.Equal(1, *a)
	s.Equal(2, *b)
}
