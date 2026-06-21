package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/app"
	"github.com/tupicapp/go-modules/concrete/worker"
	"go.uber.org/fx"
)

type WorkerSuite struct{ suite.Suite }

func TestWorkerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(WorkerSuite))
}

// captureFilter invokes the supplied worker.Filter so a test can assert it was wired.
func captureFilter(got *worker.Filter) fx.Option {
	return fx.Invoke(func(f worker.Filter) { *got = f })
}

func (s *WorkerSuite) TestNewWorkerApp_SuppliesSubscriptionsAsFilter() {
	var got worker.Filter
	a := app.NewWorkerApp(fx.Options(fx.NopLogger, captureFilter(&got)), []string{"events", "tasks"})

	s.Require().NoError(a.Start(context.Background()))
	s.Require().NoError(a.Stop(context.Background()))

	s.Equal([]string{"events", "tasks"}, got.Subscriptions)
}

func (s *WorkerSuite) TestNewWorkerApp_SuppliesEmptyFilterForAllSubscriptions() {
	var got worker.Filter
	a := app.NewWorkerApp(fx.Options(fx.NopLogger, captureFilter(&got)), nil)

	s.Require().NoError(a.Start(context.Background()))
	s.Require().NoError(a.Stop(context.Background()))

	s.Empty(got.Subscriptions)
}
