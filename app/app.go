// Package app provides shared application bootstrap helpers: startup/shutdown
// lifecycle logging and the console application runner.
package app

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
	"github.com/tupic/common-go/logger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Info identifies the running application in lifecycle logs. Supplied by the
// service from its config constants.
type Info struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}

// Lifecycle returns an fx option that logs application start and stop.
// Requires an app.Info and a logger.Logger in the graph.
func Lifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, l logger.Logger, info Info) {
		lc.Append(fx.Hook{
			OnStart: onStart(l, info),
			OnStop:  onStop(l),
		})
	})
}

func onStart(l logger.Logger, info Info) func(context.Context) error {
	return func(ctx context.Context) error {
		l.Info("app: starting...",
			zap.String("name", info.Name),
			zap.String("version", info.Version),
			zap.String("environment", info.Environment),
			zap.Bool("debug", info.Debug),
		)
		return nil
	}
}

func onStop(l logger.Logger) func(context.Context) error {
	return func(ctx context.Context) error {
		l.Info("app: stopping...")
		if err := ctx.Err(); err != nil {
			l.Warn("app: stopping exceeded the timeout", zap.Error(err))
			return nil
		}
		l.Info("app: stopped successfully")
		return nil
	}
}

// ConsoleApp runs a cobra root command inside a started fx application.
type ConsoleApp struct {
	app     *fx.App
	rootCmd *cobra.Command
}

func NewConsoleApp(modules fx.Option, rootCmd *cobra.Command) *ConsoleApp {
	fxApp := fx.New(modules, fx.Supply(rootCmd))
	return &ConsoleApp{app: fxApp, rootCmd: rootCmd}
}

func (c *ConsoleApp) Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := c.app.Start(ctx); err != nil {
		return errors.WithStack(err)
	}

	c.rootCmd.SetContext(ctx)
	c.rootCmd.SetArgs(args)

	cmdDone := make(chan error, 1)
	go func() { cmdDone <- c.rootCmd.Execute() }()

	var cmdErr error
	select {
	case cmdErr = <-cmdDone:
		// command finished normally
	case <-ctx.Done():
		// first signal: re-arm so a second signal terminates immediately
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		cmdErr = <-cmdDone
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if stopErr := c.app.Stop(stopCtx); stopErr != nil && cmdErr == nil {
		return errors.WithStack(stopErr)
	}

	return errors.WithStack(cmdErr)
}
