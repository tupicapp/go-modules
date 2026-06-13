package migrator_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/logger"
	pconfig "github.com/tupicapp/common-go/persistence/config"
	"github.com/tupicapp/common-go/persistence/connector"
	"github.com/tupicapp/common-go/persistence/migrator"
)

type MigratorSuite struct{ suite.Suite }

func TestMigratorSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MigratorSuite))
}

func (s *MigratorSuite) cfg(driver string) pconfig.Config {
	return pconfig.Config{
		Driver: driver,
		Postgres: pconfig.PostgresConfig{
			Host: "127.0.0.1", Port: 5432, Name: "tupic",
			Username: "app", Password: "x", SSLMode: "disable", Timeout: 1,
		},
	}
}

func (s *MigratorSuite) TestNew_PostgresDriver_ReturnsMigrator() {
	conn, _, err := connector.New(s.cfg("postgres"))
	s.Require().NoError(err)
	s.T().Cleanup(func() { _ = conn.DB().Close() })

	m, err := migrator.New(s.cfg("postgres"), logger.NewNoop(), conn)
	s.Require().NoError(err)
	s.NotNil(m)
}

func (s *MigratorSuite) TestNew_UnknownDriver_ReturnsError() {
	m, err := migrator.New(s.cfg("mongo"), logger.NewNoop(), nil)
	s.Require().Error(err)
	s.Nil(m)
	s.ErrorContains(err, "mongo")
}
