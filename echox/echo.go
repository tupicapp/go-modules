// Package echox provides the shared Echo HTTP stack: instance assembly,
// apperror-aware error handling, request logging, Sentry integration, the
// HTTP server lifecycle, and actor guard middlewares.
package echox

import (
	"net/http"
	"strings"

	"github.com/cockroachdb/errors"
	labecho "github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/tupic/common-go/apperror"
	"github.com/tupic/common-go/logger"
	"github.com/tupic/common-go/validator"
)

// New creates a fully configured Echo instance with the standard middleware
// stack (Sentry, CORS, Recover, request logging) plus any extra middlewares
// the service supplies (typically its authenticator).
func New(v validator.Validator, l logger.Logger, extra ...labecho.MiddlewareFunc) *labecho.Echo {
	e := labecho.New()
	e.Validator = v

	e.HTTPErrorHandler = ErrorHandler

	mws := []labecho.MiddlewareFunc{
		Sentry(),
		echoMiddleware.CORS("*"),
		echoMiddleware.Recover(),
		Logger(l),
	}
	mws = append(mws, extra...)
	e.Use(mws...)

	return e
}

// ErrorHandler maps apperror types to HTTP statuses and reports unexpected
// errors to Sentry.
func ErrorHandler(c *labecho.Context, err error) {
	if err == nil {
		return
	}

	var appError *apperror.AppError
	if errors.As(err, &appError) {
		status := http.StatusInternalServerError
		switch appError.Type {
		case apperror.TypeAuthentication:
			status = http.StatusUnauthorized
		case apperror.TypeAuthorization:
			status = http.StatusForbidden
		case apperror.TypeNotFound:
			status = http.StatusNotFound
		case apperror.TypeValidation, apperror.TypeLogic:
			status = http.StatusUnprocessableEntity
		}

		payload := map[string]any{
			"code":    appError.Code,
			"message": appError.Message,
		}
		if len(appError.Details) > 0 {
			payload["details"] = appError.Details
		}
		if writeErr := c.JSON(status, payload); writeErr != nil {
			c.Logger().Error("failed to write error response", "error", writeErr)
		}
		return
	}

	if code := labecho.StatusCode(err); code != 0 {
		msg := http.StatusText(code)
		var httpErr *labecho.HTTPError
		if errors.As(err, &httpErr) && httpErr.Message != "" {
			msg = httpErr.Message
		}
		if writeErr := c.JSON(code, map[string]any{"message": msg}); writeErr != nil {
			c.Logger().Error("failed to write error response", "error", writeErr)
		}
		return
	}

	if hub := GetSentryHub(c); hub != nil {
		hub.CaptureException(err)
	}
	c.Logger().Error("unhandled error", "error", err)
	body := map[string]any{"message": "Internal Server Error"}
	if writeErr := c.JSON(http.StatusInternalServerError, body); writeErr != nil {
		c.Logger().Error("failed to write error response", "error", writeErr)
	}
}

// BearerToken extracts the Bearer token from the Authorization header,
// returning an empty string if not present.
func BearerToken(c *labecho.Context) string {
	h := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
