package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/app"
	"go.uber.org/fx"
)

type ServerSuite struct{ suite.Suite }

func TestServerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServerSuite))
}

func (s *ServerSuite) TestNewServerApp_BuildsAppFromModules() {
	invoked := false
	a := app.NewServerApp(fx.Options(fx.NopLogger, fx.Invoke(func() { invoked = true })))

	s.Require().NoError(a.Start(context.Background()))
	s.Require().NoError(a.Stop(context.Background()))

	s.True(invoked)
}
