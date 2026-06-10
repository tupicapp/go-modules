package persistence

import (
	"context"

	"github.com/tupic/common-go/logger"
	"github.com/tupic/common-go/persistence/connector"
	"github.com/tupic/common-go/persistence/migrator"
	"github.com/tupic/common-go/persistence/uow"
	"go.uber.org/fx"
)

// Module provides the connector (with *gorm.DB), migrator, and unit of work.
// It requires a persistence.Config in the graph, supplied by the service.
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
