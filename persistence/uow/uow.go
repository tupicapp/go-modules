// Package uow provides the UnitOfWork contract and its GORM transaction
// implementation, plus the context plumbing repositories use to join an
// ambient transaction.
package uow

import (
	"context"

	"github.com/cockroachdb/errors"
	"gorm.io/gorm"
)

// UnitOfWork wraps multiple repository operations in a single atomic
// transaction.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type key struct{}

func Inject(ctx context.Context, orm *gorm.DB) context.Context {
	return context.WithValue(ctx, key{}, orm)
}

func ORM(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if orm, ok := ctx.Value(key{}).(*gorm.DB); ok {
		return orm
	}
	return fallback.WithContext(ctx)
}

type unitOfWork struct {
	orm *gorm.DB
}

func New(orm *gorm.DB) UnitOfWork {
	return &unitOfWork{orm: orm}
}

func (u *unitOfWork) Do(ctx context.Context, fn func(context.Context) error) error {
	return errors.WithStack(u.orm.WithContext(ctx).Transaction(func(t *gorm.DB) error {
		return fn(Inject(ctx, t))
	}))
}
