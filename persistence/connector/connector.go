// Package connector opens and supervises the SQL connection pool and its GORM handle.
package connector

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	_ "github.com/lib/pq"
	"github.com/tupicapp/common-go/logger"
	pconfig "github.com/tupicapp/common-go/persistence/config"
	"go.uber.org/zap"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Connector struct {
	db     *sql.DB
	driver string
}

const (
	connMaxLifetime  = 5 * time.Minute
	connMaxIdleTime  = 5 * time.Minute
	maxIdleConns     = 5
	maxOpenConns     = 15
	testMaxIdleConns = 3
	testMaxOpenConns = 3
)

func New(cfg pconfig.Config) (*Connector, *gorm.DB, error) {
	driver := cfg.Driver
	dsn := buildDSN(cfg.Postgres)

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "%s: open", driver)
	}
	configurePool(db, cfg.Test)

	gDB, err := gorm.Open(gormpg.New(gormpg.Config{Conn: db}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, nil, errors.CombineErrors(errors.Wrapf(err, "%s: gorm open", driver), db.Close())
	}

	return &Connector{db: db, driver: driver}, gDB, nil
}

func (c *Connector) Start(ctx context.Context, l logger.Logger, cfg pconfig.Config) error {
	if cfg.Test && c.db.Stats().InUse > 0 {
		return nil
	}

	timeout := time.Duration(cfg.Postgres.Timeout) * time.Second
	tCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-tCtx.Done():
			return errors.Newf("%s: connection timed out", c.driver)
		default:
			if err := c.db.PingContext(tCtx); err == nil {
				l.Debug(fmt.Sprintf("%s: connection established", c.driver))
				return nil
			} else {
				l.Debug(fmt.Sprintf("%s: waiting for connection...", c.driver), zap.Error(err))
				time.Sleep(time.Second)
			}
		}
	}
}

func (c *Connector) Stop(l logger.Logger) error {
	if err := c.db.Close(); err != nil {
		return errors.Wrap(err, "db close")
	}
	l.Debug(fmt.Sprintf("%s: connection closed", c.driver))
	return nil
}

func (c *Connector) DB() *sql.DB { return c.db }

func buildDSN(pg pconfig.PostgresConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		pg.Host, pg.Port, pg.Name, pg.Username, pg.Password, pg.SSLMode,
	)
}

func configurePool(db *sql.DB, test bool) {
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)
	if test {
		db.SetMaxIdleConns(testMaxIdleConns)
		db.SetMaxOpenConns(testMaxOpenConns)
		return
	}
	db.SetMaxIdleConns(maxIdleConns)
	db.SetMaxOpenConns(maxOpenConns)
}
