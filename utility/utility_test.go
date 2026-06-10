package utility_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/tupic/common-go/utility"
)

type UtilitySuite struct{ suite.Suite }

func TestUtilitySuite(t *testing.T) {
	suite.Run(t, new(UtilitySuite))
}

func (s *UtilitySuite) TestStringOrNil_NonEmptyString_ReturnsPointer() {
	got := utility.StringOrNil("hello")
	s.Require().NotNil(got)
	s.Equal("hello", *got)
}

func (s *UtilitySuite) TestStringOrNil_TrimsSpaces() {
	got := utility.StringOrNil("  spaces  ")
	s.Require().NotNil(got)
	s.Equal("spaces", *got)
}

func (s *UtilitySuite) TestStringOrNil_EmptyString_ReturnsNil() {
	s.Nil(utility.StringOrNil(""))
}

func (s *UtilitySuite) TestStringOrNil_WhitespaceOnly_ReturnsNil() {
	s.Nil(utility.StringOrNil("   "))
}

func (s *UtilitySuite) TestStringDeref_NilPointer_ReturnsEmpty() {
	s.Equal("", utility.StringDereference(nil))
}

func (s *UtilitySuite) TestStringDeref_NonNilPointer_ReturnsValue() {
	v := "hello"
	s.Equal("hello", utility.StringDereference(&v))
}

func (s *UtilitySuite) TestStringDeref_PointerToEmpty_ReturnsEmpty() {
	v := ""
	s.Equal("", utility.StringDereference(&v))
}

func (s *UtilitySuite) TestParseUUID_Valid() {
	id, err := uuid.NewV7()
	s.Require().NoError(err)

	got, err := utility.ParseUUID(id.String())
	s.Require().NoError(err)
	s.Equal(id, got)
}

func (s *UtilitySuite) TestParseUUID_Invalid_ReturnsError() {
	_, err := utility.ParseUUID("not-a-uuid")
	s.Require().Error(err)
}
