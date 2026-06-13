package validator_test

import (
	"testing"

	playgroundLib "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/apperror"
	"github.com/tupicapp/common-go/validator"
)

type signup struct {
	Email  string `validate:"required,email"`
	Locale string `validate:"locale"`
}

type ValidatorSuite struct{ suite.Suite }

func TestValidatorSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ValidatorSuite))
}

// fields flattens an AppError's validation details into field→message.
func (s *ValidatorSuite) fields(err error) map[string]string {
	var ae *apperror.AppError
	s.Require().ErrorAs(err, &ae)
	s.True(ae.IsValidation())

	out := map[string]string{}
	for _, d := range ae.Details {
		record, ok := d.(map[string]interface{})
		s.Require().True(ok)
		out[record["field"].(string)] = record["message"].(string)
	}
	return out
}

func (s *ValidatorSuite) TestValidate_ValidStruct_ReturnsNil() {
	err := validator.NewPlayground().Validate(signup{Email: "a@b.com", Locale: "en-US"})
	s.NoError(err)
}

func (s *ValidatorSuite) TestValidate_RequiredField_ReturnsValidationError() {
	err := validator.NewPlayground().Validate(signup{Locale: "en-US"})

	fields := s.fields(err)
	s.Equal("Email is required.", fields["Email"])
}

func (s *ValidatorSuite) TestValidate_InvalidEmail_ReturnsEmailMessage() {
	err := validator.NewPlayground().Validate(signup{Email: "not-an-email", Locale: "en-US"})

	fields := s.fields(err)
	s.Equal("Email must be a valid email address.", fields["Email"])
}

func (s *ValidatorSuite) TestValidate_InvalidLocale_ReturnsLocaleMessage() {
	err := validator.NewPlayground().Validate(signup{Email: "a@b.com", Locale: "english"})

	fields := s.fields(err)
	s.Equal("Locale must be a valid locale code (e.g. en-US).", fields["Locale"])
}

func (s *ValidatorSuite) TestValidate_ReportsEveryViolation() {
	err := validator.NewPlayground().Validate(signup{})

	fields := s.fields(err)
	s.Len(fields, 2)
	s.Contains(fields, "Email")
	s.Contains(fields, "Locale")
}

func (s *ValidatorSuite) TestWithAlias_RegistersTagAlias() {
	type form struct {
		Slug string `validate:"slug"`
	}
	v := validator.NewPlayground(validator.WithAlias("slug", "required,min=3"))

	fields := s.fields(v.Validate(form{Slug: "ab"}))
	s.Equal("Slug must be at least 3 characters long.", fields["Slug"])
}

func (s *ValidatorSuite) TestWithValidation_RegistersCustomTag() {
	type form struct {
		N int `validate:"even"`
	}
	v := validator.NewPlayground(validator.WithValidation("even", func(fl playgroundLib.FieldLevel) bool {
		return fl.Field().Int()%2 == 0
	}))

	s.NoError(v.Validate(form{N: 4}))
	s.Require().Error(v.Validate(form{N: 3}))
}
