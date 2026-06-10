// Package postgres implements the migration contract with golang-migrate.
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/tupic/common-go/logger"
	pconfig "github.com/tupic/common-go/persistence/config"
	"github.com/tupic/common-go/persistence/connector"
	"github.com/tupic/common-go/persistence/migrator/contract"
	"go.uber.org/zap"
)

type Migrator struct {
	db            *sql.DB
	migrationPath string
	dbName        string
	logger        logger.Logger
}

func New(cfg pconfig.Config, l logger.Logger, c *connector.Connector) contract.Migrator {
	return &Migrator{
		db:            c.DB(),
		migrationPath: cfg.Postgres.MigrationPath,
		dbName:        cfg.Postgres.Name,
		logger:        l,
	}
}

func (m *Migrator) instance() (*migrate.Migrate, error) {
	driver, err := migratepostgres.WithInstance(m.db, &migratepostgres.Config{DatabaseName: m.dbName})
	if err != nil {
		return nil, errors.Wrap(err, "migrator: create driver")
	}
	mg, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.migrationPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, errors.Wrap(err, "migrator: create instance")
	}
	return mg, nil
}

func (m *Migrator) Status(_ context.Context) (*contract.Status, error) {
	mg, err := m.instance()
	if err != nil {
		return nil, err
	}
	version, dirty, err := mg.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return &contract.Status{HasVersion: false}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "migrator: status")
	}
	return &contract.Status{Version: version, HasVersion: true, Dirty: dirty}, nil
}

func (m *Migrator) Migrate(_ context.Context) error {
	mg, err := m.instance()
	if err != nil {
		return err
	}
	if err = mg.Up(); errors.Is(err, migrate.ErrNoChange) {
		m.logger.Info("migrator: no new migrations")
		return nil
	}
	return errors.Wrap(err, "migrator: up")
}

func (m *Migrator) Rollback(_ context.Context) error {
	mg, err := m.instance()
	if err != nil {
		return err
	}
	return errors.Wrap(mg.Steps(-1), "migrator: rollback")
}

func (m *Migrator) Fresh(ctx context.Context) error {
	mg, err := m.instance()
	if err != nil {
		return err
	}
	m.logger.Info("migrator: dropping all tables", zap.String("db", m.dbName))
	if dropErr := mg.Drop(); dropErr != nil && !errors.Is(dropErr, migrate.ErrNoChange) {
		return errors.Wrap(dropErr, "migrator: drop")
	}
	return m.Migrate(ctx)
}
