package uow_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/persistence/uow"
	"gorm.io/gorm"
)

type UoWSuite struct{ suite.Suite }

func TestUoWSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UoWSuite))
}

func (s *UoWSuite) TestORM_ReturnsInjectedTransaction() {
	tx := &gorm.DB{}
	fallback := &gorm.DB{}
	ctx := uow.Inject(context.Background(), tx)

	// When a transaction is present in context, ORM returns it verbatim and never consults the fallback.
	s.Same(tx, uow.ORM(ctx, fallback))
}

func (s *UoWSuite) TestORM_PrefersInjectedOverFallback() {
	first := &gorm.DB{}
	second := &gorm.DB{}

	// The most recent injection shadows any earlier one.
	ctx := uow.Inject(uow.Inject(context.Background(), first), second)
	s.Same(second, uow.ORM(ctx, &gorm.DB{}))
}
