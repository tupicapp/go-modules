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

func (m *Migrator) instance(ctx context.Context) (*migrate.Migrate, error) {
	// Build the driver on a dedicated connection (not the whole pool) so
	// closing the migrate instance releases exactly that connection.
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "migrator: acquire connection")
	}
	driver, err := migratepostgres.WithConnection(ctx, conn, &migratepostgres.Config{DatabaseName: m.dbName})
	if err != nil {
		_ = conn.Close()
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

// close releases the migrate instance's dedicated database connection back to
// the pool. Without this every operation leaks one pooled connection, which
// deadlocks small test pools (e.g. Fresh on suite teardown blocking forever).
func (m *Migrator) close(mg *migrate.Migrate) {
	srcErr, dbErr := mg.Close()
	if srcErr != nil || dbErr != nil {
		m.logger.Warn("migrator: close failed",
			zap.NamedError("source", srcErr),
			zap.NamedError("db", dbErr),
		)
	}
}

func (m *Migrator) Status(ctx context.Context) (*contract.Status, error) {
	mg, err := m.instance(ctx)
	if err != nil {
		return nil, err
	}
	defer m.close(mg)
	version, dirty, err := mg.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return &contract.Status{HasVersion: false}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "migrator: status")
	}
	return &contract.Status{Version: version, HasVersion: true, Dirty: dirty}, nil
}

func (m *Migrator) Migrate(ctx context.Context) error {
	mg, err := m.instance(ctx)
	if err != nil {
		return err
	}
	defer m.close(mg)
	if err = mg.Up(); errors.Is(err, migrate.ErrNoChange) {
		m.logger.Info("migrator: no new migrations")
		return nil
	}
	return errors.Wrap(err, "migrator: up")
}

func (m *Migrator) Rollback(ctx context.Context) error {
	mg, err := m.instance(ctx)
	if err != nil {
		return err
	}
	defer m.close(mg)
	return errors.Wrap(mg.Steps(-1), "migrator: rollback")
}

func (m *Migrator) Fresh(ctx context.Context) error {
	mg, err := m.instance(ctx)
	if err != nil {
		return err
	}
	m.logger.Info("migrator: dropping all tables", zap.String("db", m.dbName))
	if dropErr := mg.Drop(); dropErr != nil && !errors.Is(dropErr, migrate.ErrNoChange) {
		m.close(mg)
		return errors.Wrap(dropErr, "migrator: drop")
	}
	// Release the drop instance's connection before Migrate acquires its own.
	m.close(mg)
	return m.Migrate(ctx)
}
