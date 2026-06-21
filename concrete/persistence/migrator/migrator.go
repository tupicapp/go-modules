// Package migrator selects the migration driver from configuration.
package migrator

import (
	"github.com/cockroachdb/errors"
	pconfig "github.com/tupicapp/go-modules/concrete/persistence/config"
	"github.com/tupicapp/go-modules/concrete/persistence/connector"
	"github.com/tupicapp/go-modules/concrete/persistence/migrator/contract"
	"github.com/tupicapp/go-modules/concrete/persistence/migrator/postgres"
	"github.com/tupicapp/go-modules/contract/logger"
)

func New(cfg pconfig.Config, l logger.Logger, c *connector.Connector) (contract.Migrator, error) {
	switch cfg.Driver {
	case "postgres":
		return postgres.New(cfg, l, c), nil
	default:
		return nil, errors.Newf("persistence: unknown driver: %s", cfg.Driver)
	}
}
