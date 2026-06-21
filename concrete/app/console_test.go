package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/app"
	"go.uber.org/fx"
)

type ConsoleSuite struct{ suite.Suite }

func TestConsoleSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ConsoleSuite))
}

// cmd builds a root command whose RunE delegates to fn (with output silenced).
func (s *ConsoleSuite) cmd(fn func() error) *cobra.Command {
	return &cobra.Command{
		Use:           "root",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          func(_ *cobra.Command, _ []string) error { return fn() },
	}
}

func (s *ConsoleSuite) TestRun_ExecutesCommand() {
	ran := false
	c := app.NewConsoleApp(fx.NopLogger, s.cmd(func() error { ran = true; return nil }))

	s.Require().NoError(c.Run([]string{}))
	s.True(ran)
}

func (s *ConsoleSuite) TestRun_PropagatesCommandError() {
	wantErr := errors.New("boom")
	c := app.NewConsoleApp(fx.NopLogger, s.cmd(func() error { return wantErr }))

	err := c.Run(nil)
	s.Require().Error(err)
	s.ErrorIs(err, wantErr) // WithStack must preserve the wrapped sentinel
}

func (s *ConsoleSuite) TestRun_ReturnsStartErrorBeforeCommand() {
	ran := false
	// A fx.Invoke that fails makes app.Start error, so the command never runs.
	mods := fx.Options(fx.NopLogger, fx.Invoke(func() error { return errors.New("start failed") }))
	c := app.NewConsoleApp(mods, s.cmd(func() error { ran = true; return nil }))

	s.Require().Error(c.Run(nil))
	s.False(ran)
}

func (s *ConsoleSuite) TestRun_ReturnsPanicAsError() {
	c := app.NewConsoleApp(fx.NopLogger, s.cmd(func() error { panic("boom") }))

	err := c.Run(nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "console command panic: boom")
}

func (s *ConsoleSuite) TestRun_StopsAppAfterPanic() {
	stopped := false
	mods := fx.Options(
		fx.NopLogger,
		fx.Invoke(func(lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStop: func(context.Context) error {
					stopped = true
					return nil
				},
			})
		}),
	)
	c := app.NewConsoleApp(mods, s.cmd(func() error { panic("boom") }))

	s.Require().Error(c.Run(nil))
	s.True(stopped)
}
