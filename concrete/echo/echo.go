// Package echo provides the shared Echo HTTP stack: instance assembly, apperror-aware error handling, request logging,
// Sentry integration, the HTTP server lifecycle, and actor guard middlewares.
package echo

import (
	"context"
	"net/http"

	"github.com/cockroachdb/errors"
	base "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/tupicapp/go-modules/contract/authentication"
	"github.com/tupicapp/go-modules/contract/logger"
	"github.com/tupicapp/go-modules/contract/validator"
	"github.com/tupicapp/go-modules/kernel/apperror"
)

// statusClientClosedRequest mirrors nginx's 499: the client closed the connection before the server response.
const statusClientClosedRequest = 499

// NewEcho creates a fully configured Echo instance with the standard middleware stack.
func NewEcho[U any](v validator.Validator, l logger.Logger, a authentication.Authenticator[U]) *base.Echo {
	e := base.New()
	e.Validator = v
	e.HTTPErrorHandler = HTTPErrorHandler
	e.Use([]base.MiddlewareFunc{
		Sentry(),
		Logger(l),
		middleware.CORS("*"),
		middleware.Recover(),
		Authenticator(a),
	}...)
	return e
}

// HTTPErrorHandler maps apperror types to HTTP statuses and reports unexpected errors to Sentry.
func HTTPErrorHandler(c *base.Context, err error) {
	if err == nil {
		return
	}

	// A canceled request context means the client disconnected mid-request.
	// This is not a server fault, so respond with 499 and skip Sentry to avoid noise.
	if errors.Is(err, context.Canceled) {
		if writeErr := c.NoContent(statusClientClosedRequest); writeErr != nil {
			c.Logger().Debug("client closed request", "error", err)
		}
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

	if code := base.StatusCode(err); code != 0 {
		msg := http.StatusText(code)
		var httpErr *base.HTTPError
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
