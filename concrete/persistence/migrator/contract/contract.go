// Package contract defines the migration driver contract.
package contract

import "context"

type Status struct {
	Version    uint
	HasVersion bool
	Dirty      bool
}

type Migrator interface {
	Status(ctx context.Context) (*Status, error)
	Migrate(ctx context.Context) error
	Rollback(ctx context.Context) error
	Fresh(ctx context.Context) error
}
