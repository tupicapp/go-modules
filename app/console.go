package app

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// ConsoleApp runs an app to run as CLI using cobra root command inside a started fx application.
type ConsoleApp struct {
	app     *fx.App
	command *cobra.Command
}

func NewConsoleApp(modules fx.Option, rootCmd *cobra.Command) *ConsoleApp {
	fxApp := fx.New(modules, fx.Supply(rootCmd))
	return &ConsoleApp{app: fxApp, command: rootCmd}
}

func (c *ConsoleApp) Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := c.app.Start(ctx); err != nil {
		return errors.WithStack(err)
	}

	c.command.SetContext(ctx)
	c.command.SetArgs(args)

	done := make(chan error, 1)
	go func() { done <- c.command.Execute() }()

	var err error
	select {
	case err = <-done:
		// command finished normally
	case <-ctx.Done():
		// first signal: re-arm so a second signal terminates immediately
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		err = <-done
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if stopErr := c.app.Stop(stopCtx); stopErr != nil && err == nil {
		return errors.WithStack(stopErr)
	}

	return errors.WithStack(err)
}
