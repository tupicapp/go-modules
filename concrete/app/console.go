package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// ConsoleApp runs an app to run as CLI using cobra root command inside a started fx application.
type ConsoleApp struct {
	app     *fx.App
	command *cobra.Command
}

type consoleResult struct {
	err      error
	captured bool
}

const consoleSentryFlushTimeout = 2 * time.Second

func NewConsoleApp(modules fx.Option, rootCmd *cobra.Command) *ConsoleApp {
	fxApp := fx.New(modules, fx.Supply(rootCmd))
	return &ConsoleApp{app: fxApp, command: rootCmd}
}

func (c *ConsoleApp) Run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := c.app.Start(ctx); err != nil {
		captureConsoleError(err)
		flushConsoleSentry()
		return errors.WithStack(err)
	}

	c.command.SetContext(ctx)
	c.command.SetArgs(args)

	done := make(chan consoleResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				captureConsolePanic(r)
				done <- consoleResult{err: consolePanicError(r), captured: true}
			}
		}()

		done <- consoleResult{err: c.command.Execute()}
	}()

	var result consoleResult
	select {
	case result = <-done:
		// command finished normally
	case <-ctx.Done():
		// first signal: re-arm so a second signal terminates immediately
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		result = <-done
	}

	if result.err != nil && !result.captured {
		captureConsoleError(result.err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if stopErr := c.app.Stop(stopCtx); stopErr != nil {
		captureConsoleError(stopErr)
		flushConsoleSentry()
		if result.err == nil {
			return errors.WithStack(stopErr)
		}
	}

	return errors.WithStack(result.err)
}

func captureConsoleError(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		sentry.CaptureException(err)
	}
}

func captureConsolePanic(v any) {
	sentry.CurrentHub().Recover(v)
}

func flushConsoleSentry() {
	sentry.Flush(consoleSentryFlushTimeout)
}

func consolePanicError(v any) error {
	if err, ok := v.(error); ok {
		return errors.Wrap(err, "console command panic")
	}
	return errors.Newf("console command panic: %s", fmt.Sprint(v))
}
