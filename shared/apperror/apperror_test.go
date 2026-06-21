package apperror_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/shared/apperror"
)

type AppErrorSuite struct{ suite.Suite }

func TestAppErrorSuite(t *testing.T) {
	suite.Run(t, new(AppErrorSuite))
}

func (s *AppErrorSuite) TestTypeHelpers_Logic() {
	err := apperror.Logic("oops", "domain_code")
	s.True(err.IsLogic())
	s.False(err.IsValidation())
	s.False(err.IsNotFound())
	s.False(err.IsAuthentication())
	s.False(err.IsAuthorization())
}

func (s *AppErrorSuite) TestTypeHelpers_Validation() {
	err := apperror.Validation(map[string]string{})
	s.True(err.IsValidation())
}

func (s *AppErrorSuite) TestTypeHelpers_NotFound() {
	err := apperror.NotFound("missing")
	s.True(err.IsNotFound())
}

func (s *AppErrorSuite) TestTypeHelpers_Authentication() {
	err := apperror.Authentication("bad token")
	s.True(err.IsAuthentication())
}

func (s *AppErrorSuite) TestTypeHelpers_Authorization() {
	err := apperror.Authorization("nope")
	s.True(err.IsAuthorization())
}

func (s *AppErrorSuite) TestNewValidationError_Metadata() {
	err := apperror.Validation(map[string]string{
		"Value": "invalid",
		"Name":  "required",
	})

	s.Len(err.Details, 2)

	record, ok := err.Details[0].(map[string]interface{})
	s.Require().True(ok)
	s.Contains(record, "field")
	s.Contains(record, "message")
	s.Equal(apperror.CodeValidation, err.Code)
}

func (s *AppErrorSuite) TestIsAppError_WrappedAppError() {
	err := apperror.NotFound("boom")
	wrapped := &wrappedError{err: err}
	s.True(apperror.IsAppError(wrapped))
}

func (s *AppErrorSuite) TestIsAppError_PlainError() {
	s.False(apperror.IsAppError(errors.New("plain error")))
}

// wrappedError wraps an error to simulate deep wrapping.
type wrappedError struct{ err error }

func (w *wrappedError) Error() string { return w.err.Error() }
func (w *wrappedError) Unwrap() error { return w.err }
