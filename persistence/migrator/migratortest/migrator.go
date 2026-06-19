// Package migratortest provides recording test doubles for the migrator contract.
package migratortest

import (
	"context"

	"github.com/tupicapp/go-modules/persistence/migrator/contract"
)

// Migrator is a spy migrator that only records which operations were invoked.
type Migrator struct {
	MigrateCalled  bool
	RollbackCalled bool
	FreshCalled    bool
	State          *contract.Status
}

func (m *Migrator) Migrate(context.Context) error  { m.MigrateCalled = true; return nil }
func (m *Migrator) Rollback(context.Context) error { m.RollbackCalled = true; return nil }
func (m *Migrator) Fresh(context.Context) error    { m.FreshCalled = true; return nil }
func (m *Migrator) Status(context.Context) (*contract.Status, error) {
	return m.State, nil
}

var _ contract.Migrator = (*Migrator)(nil)
