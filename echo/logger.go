package echo

import (
	"fmt"
	"net/http"
	"time"

	labecho "github.com/labstack/echo/v5"
	"github.com/tupicapp/go-modules/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger emits structured logs for every HTTP request.
func Logger(l logger.Logger) labecho.MiddlewareFunc {
	return func(next labecho.HandlerFunc) labecho.HandlerFunc {
		return func(ctx *labecho.Context) error {
			start := time.Now()

			err := next(ctx)
			if err != nil {
				ctx.Echo().HTTPErrorHandler(ctx, err)
			}

			req := ctx.Request()
			resp, status := labecho.ResolveResponseStatus(ctx.Response(), err)

			var size int64
			if resp != nil {
				size = resp.Size
			}

			fields := []zapcore.Field{
				zap.String("remote_ip", ctx.RealIP()),
				zap.String("latency", time.Since(start).String()),
				zap.String("host", req.Host),
				zap.String("request", fmt.Sprintf("%s %s", req.Method, req.RequestURI)),
				zap.Int("status", status),
				zap.Int64("size", size),
				zap.String("user_agent", req.UserAgent()),
			}

			if id := requestID(req, ctx.Response()); id != "" {
				fields = append(fields, zap.String("request_id", id))
			}

			switch {
			case status >= 500:
				l.Error("server error", append(fields, zap.Error(err))...)
			case status >= 400:
				l.Debug("client error", append(fields, zap.Error(err))...)
			case status >= 300:
				l.Debug("redirection", fields...)
			default:
				l.Debug("success", fields...)
			}

			return nil
		}
	}
}

func requestID(req *http.Request, res http.ResponseWriter) string {
	if id := req.Header.Get(labecho.HeaderXRequestID); id != "" {
		return id
	}
	return res.Header().Get(labecho.HeaderXRequestID)
}
