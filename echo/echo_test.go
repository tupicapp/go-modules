package echo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cockroachdb/errors"
	labecho "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/apperror"
)

type ErrorHandlerSuite struct {
	suite.Suite
	e *labecho.Echo
}

func TestErrorHandlerSuite(t *testing.T) {
	suite.Run(t, new(ErrorHandlerSuite))
}

func (s *ErrorHandlerSuite) SetupTest() {
	s.e = labecho.New()
}

func (s *ErrorHandlerSuite) invoke(err error) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	ErrorHandler(s.e.NewContext(req, rec), err)
	return rec
}

func (s *ErrorHandlerSuite) body(rec *httptest.ResponseRecorder) map[string]any {
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body
}

func (s *ErrorHandlerSuite) TestNilError_WritesNothing() {
	s.Empty(s.invoke(nil).Body.String())
}

func (s *ErrorHandlerSuite) TestAppErrors_StatusCodes() {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"authentication", apperror.Authentication("x"), http.StatusUnauthorized},
		{"authorization", apperror.Authorization("x"), http.StatusForbidden},
		{"not found", apperror.NotFound("x"), http.StatusNotFound},
		{"validation", apperror.Validation(map[string]string{"f": "required"}), http.StatusUnprocessableEntity},
		{"echo not found", labecho.ErrNotFound, http.StatusNotFound},
		{"echo internal error", labecho.ErrInternalServerError, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		s.Equal(tc.status, s.invoke(tc.err).Code, tc.name)
	}
}

func (s *ErrorHandlerSuite) TestAuthentication_IncludesErrorCode() {
	rec := s.invoke(apperror.Authentication("x"))
	s.Equal(string(apperror.CodeAuthentication), s.body(rec)["code"])
}

func (s *ErrorHandlerSuite) TestValidation_IncludesDetails() {
	rec := s.invoke(apperror.Validation(map[string]string{"name": "required"}))
	s.NotEmpty(s.body(rec)["details"])
}

func (s *ErrorHandlerSuite) TestHTTPErrors_IncludeMessage() {
	cases := []struct {
		err     error
		message string
	}{
		{labecho.NewHTTPError(http.StatusNotFound, "Not Found"), "Not Found"},
		{labecho.NewHTTPError(http.StatusInternalServerError, "Internal Server Error"), "Internal Server Error"},
	}
	for _, tc := range cases {
		s.Equal(tc.message, s.body(s.invoke(tc.err))["message"])
	}
}

func (s *ErrorHandlerSuite) TestCanceledContext_Returns499WithoutBody() {
	// Both a bare and a wrapped context.Canceled must be treated as a client
	// disconnect: 499, empty body, no Sentry capture.
	for _, err := range []error{context.Canceled, errors.Wrap(context.Canceled, "reading body")} {
		rec := s.invoke(err)
		s.Equal(statusClientClosedRequest, rec.Code)
		s.Empty(rec.Body.String())
	}
}
