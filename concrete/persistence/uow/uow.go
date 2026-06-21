// Package uow provides the GORM implementation of the UnitOfWork contract, plus the context plumbing repositories use
// to join an ambient transaction.
package uow

import (
	"context"

	"github.com/cockroachdb/errors"
	contract "github.com/tupicapp/go-modules/contract/persistence"
	"gorm.io/gorm"
)

type key struct{}

func Inject(ctx context.Context, orm *gorm.DB) context.Context {
	return context.WithValue(ctx, key{}, orm)
}

// ORM returns the ambient transaction DB injected by Do, falling back to the supplied db when none is present.
func ORM(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if orm, ok := ctx.Value(key{}).(*gorm.DB); ok {
		return orm
	}
	return fallback.WithContext(ctx)
}

type unitOfWork struct {
	orm *gorm.DB
}

func New(orm *gorm.DB) contract.UnitOfWork {
	return &unitOfWork{orm: orm}
}

func (u *unitOfWork) Do(ctx context.Context, fn func(context.Context) error) error {
	return errors.WithStack(u.orm.WithContext(ctx).Transaction(func(t *gorm.DB) error {
		return fn(Inject(ctx, t))
	}))
}

var _ contract.UnitOfWork = (*unitOfWork)(nil)
