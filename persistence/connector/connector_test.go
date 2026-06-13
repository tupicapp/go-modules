package connector

import (
	"testing"

	"github.com/stretchr/testify/suite"
	pconfig "github.com/tupicapp/common-go/persistence/config"
)

type ConnectorSuite struct{ suite.Suite }

func TestConnectorSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ConnectorSuite))
}

func (s *ConnectorSuite) TestBuildDSN_FormatsAllFields() {
	dsn := buildDSN(pconfig.PostgresConfig{
		Host:     "db.internal",
		Port:     5432,
		Name:     "tupic",
		Username: "app",
		Password: "s3cret",
		SSLMode:  "require",
	})

	s.Equal("host=db.internal port=5432 dbname=tupic user=app password=s3cret sslmode=require", dsn)
}

func (s *ConnectorSuite) TestNew_OpensWithoutConnecting() {
	// sql.Open for postgres is lazy and DisableAutomaticPing keeps gorm from dialing, so New succeeds against a
	// non-existent host.
	c, db, err := New(pconfig.Config{
		Driver: "postgres",
		Postgres: pconfig.PostgresConfig{
			Host: "127.0.0.1", Port: 5432, Name: "tupic",
			Username: "app", Password: "x", SSLMode: "disable", Timeout: 1,
		},
	})
	s.Require().NoError(err)
	s.Require().NotNil(c)
	s.NotNil(db)
	s.NotNil(c.DB())
	s.Require().NoError(c.DB().Close())
}
