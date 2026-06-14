// Package migrator selects the migration driver from configuration.
package migrator

import (
	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/logger"
	pconfig "github.com/tupicapp/go-modules/persistence/config"
	"github.com/tupicapp/go-modules/persistence/connector"
	"github.com/tupicapp/go-modules/persistence/migrator/contract"
	"github.com/tupicapp/go-modules/persistence/migrator/postgres"
)

func New(cfg pconfig.Config, l logger.Logger, c *connector.Connector) (contract.Migrator, error) {
	switch cfg.Driver {
	case "postgres":
		return postgres.New(cfg, l, c), nil
	default:
		return nil, errors.Newf("persistence: unknown driver: %s", cfg.Driver)
	}
}
