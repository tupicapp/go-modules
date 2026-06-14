package persistence

import (
	"context"

	"github.com/tupicapp/go-modules/logger"
	"github.com/tupicapp/go-modules/persistence/connector"
	"github.com/tupicapp/go-modules/persistence/migrator"
	"github.com/tupicapp/go-modules/persistence/uow"
	"go.uber.org/fx"
)

// Module provides the connector (with *gorm.DB), migrator, and unit of work. It requires a persistence.Config in the
// graph, supplied by the service.
var Module = fx.Options(
	fx.Provide(connector.New, migrator.New, uow.New),
	fx.Invoke(registerLifecycle),
)

func registerLifecycle(lc fx.Lifecycle, d *connector.Connector, cfg Config, l logger.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error { return d.Start(ctx, l, cfg) },
		OnStop:  func(_ context.Context) error { return d.Stop(l) },
	})
}
