// Package migrator selects the migration driver from configuration.
package migrator

import (
	"github.com/cockroachdb/errors"
	"github.com/tupic/common-go/logger"
	pconfig "github.com/tupic/common-go/persistence/config"
	"github.com/tupic/common-go/persistence/connector"
	"github.com/tupic/common-go/persistence/migrator/contract"
	"github.com/tupic/common-go/persistence/migrator/postgres"
)

func New(cfg pconfig.Config, l logger.Logger, c *connector.Connector) (contract.Migrator, error) {
	switch cfg.Driver {
	case "postgres":
		return postgres.New(cfg, l, c), nil
	default:
		return nil, errors.Newf("persistence: unknown driver: %s", cfg.Driver)
	}
}
