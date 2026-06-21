package echo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	labecho "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/echo"
	"github.com/tupicapp/go-modules/testkit/loggertest"
)

type LoggerMiddlewareSuite struct {
	suite.Suite
	e      *labecho.Echo
	logger *loggertest.Memory
}

func TestLoggerMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(LoggerMiddlewareSuite))
}

func (s *LoggerMiddlewareSuite) SetupTest() {
	s.e = labecho.New()
	s.logger = loggertest.NewMemory()
}

func (s *LoggerMiddlewareSuite) request(handler labecho.HandlerFunc) *httptest.ResponseRecorder {
	s.e = labecho.New()
	s.e.Use(echo.Logger(s.logger))
	s.e.GET("/", handler)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

func (s *LoggerMiddlewareSuite) TestSuccess_LogsDebug() {
	rec := s.request(func(c *labecho.Context) error { return c.NoContent(http.StatusOK) })

	s.Equal(http.StatusOK, rec.Code)
	s.Require().Len(s.logger.Entries(), 1)
	s.Equal("debug", s.logger.Entries()[0].Level)
	s.Equal("success", s.logger.Entries()[0].Message)
}

func (s *LoggerMiddlewareSuite) TestClientError_LogsDebug() {
	rec := s.request(func(c *labecho.Context) error { return labecho.ErrUnauthorized })

	s.Equal(http.StatusUnauthorized, rec.Code)
	s.Require().Len(s.logger.Entries(), 1)
	s.Equal("debug", s.logger.Entries()[0].Level)
	s.Equal("client error", s.logger.Entries()[0].Message)
}

func (s *LoggerMiddlewareSuite) TestServerError_LogsError() {
	rec := s.request(func(c *labecho.Context) error { return labecho.ErrInternalServerError })

	s.Equal(http.StatusInternalServerError, rec.Code)
	s.Require().Len(s.logger.Entries(), 1)
	s.Equal("error", s.logger.Entries()[0].Level)
	s.Equal("server error", s.logger.Entries()[0].Message)
}

func (s *LoggerMiddlewareSuite) TestRedirect_LogsDebug() {
	rec := s.request(func(c *labecho.Context) error { return c.Redirect(http.StatusFound, "/other") })

	s.Equal(http.StatusFound, rec.Code)
	s.Require().Len(s.logger.Entries(), 1)
	s.Equal("debug", s.logger.Entries()[0].Level)
	s.Equal("redirection", s.logger.Entries()[0].Message)
}
