// Package uow provides the UnitOfWork contract. The GORM implementation and the context plumbing repositories use to
// join an ambient transaction live in concrete/persistence/uow.
package uow

import "context"

// UnitOfWork wraps multiple repository operations in a single atomic transaction.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
