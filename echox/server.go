package echox

import (
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	labecho "github.com/labstack/echo/v5"
	"github.com/tupic/common-go/logger"
	"go.uber.org/zap"
)

// ServerConfig holds HTTP server listen addresses and timeouts (seconds).
type ServerConfig struct {
	Hosts             []string
	ReadTimeout       int
	WriteTimeout      int
	ReadHeaderTimeout int
	IdleTimeout       int
}

type Server struct {
	servers []*http.Server
	logger  logger.Logger
}

func NewServer(e *labecho.Echo, c ServerConfig, l logger.Logger) *Server {
	servers := make([]*http.Server, 0, len(c.Hosts))
	for _, addr := range c.Hosts {
		servers = append(servers, &http.Server{
			Addr:              addr,
			Handler:           e,
			ReadTimeout:       time.Duration(c.ReadTimeout) * time.Second,
			WriteTimeout:      time.Duration(c.WriteTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(c.ReadHeaderTimeout) * time.Second,
			IdleTimeout:       time.Duration(c.IdleTimeout) * time.Second,
		})
	}
	return &Server{servers: servers, logger: l}
}

func (s *Server) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.WithStack(ctx.Err())
	}
	for _, srv := range s.servers {
		s.logger.Info("http server: starting...", zap.String("listen", srv.Addr))
		go func(srv *http.Server) {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error("http server: cannot start", zap.String("listen", srv.Addr), zap.Error(err))
			}
		}(srv)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	for _, srv := range s.servers {
		if err := srv.Shutdown(ctx); err != nil {
			return errors.Wrap(err, "cannot stop http server")
		}
	}
	s.logger.Debug("http server: stopped successfully")
	return nil
}
