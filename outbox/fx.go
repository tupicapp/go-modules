package outbox

import (
	"go.uber.org/fx"
)

// StorageModule provides the Storage and binds it as the Outbox port. Include in every app mode — server, worker,
// console — so use cases can always write to the outbox transactionally.
var StorageModule = fx.Options(
	fx.Provide(
		NewStorage,
		func(p *Storage) Outbox { return p },
	),
)

// RelayModule provides the Relay and registers its lifecycle hooks. Include only in long-running processes (worker, e2e
// tests). Integration tests must NOT include this — the relay polls the DB in a background goroutine and canceling its
// context during test teardown corrupts the shared connection. Requires an outbox.Config in the graph, supplied by the
// service.
var RelayModule = fx.Options(
	fx.Provide(NewRelay),
	fx.Invoke(func(lc fx.Lifecycle, r *Relay) {
		lc.Append(fx.Hook{OnStart: r.Start, OnStop: r.Stop})
	}),
)
