package nats

import (
	"context"

	natsLib "github.com/nats-io/nats.go"
	"go.uber.org/fx"
)

// ConnectionModule provides the NATS connection and JetStreamContext. Requires a nats.Config in the graph, supplied by
// the service.
var ConnectionModule = fx.Options(
	fx.Provide(NewConnection),
	fx.Invoke(RegisterConnectionLifecycle),
)

// SubscriberModule wires the shared Router, the EventSubscriber (non-queue.* subjects), and exposes the Router as a
// MessageHandlerRegisterer.
var SubscriberModule = fx.Options(
	fx.Provide(
		NewRouter,
		NewEventSubscriber,
		func(r *Router) MessageHandlerRegisterer { return r },
	),
	fx.Invoke(RegisterSubscriberLifecycle),
)

// WorkerModule wires the QueueSubscriber (queue.* subjects). Requires the shared Router already provided by
// SubscriberModule.
var WorkerModule = fx.Options(
	fx.Provide(NewQueueSubscriber),
	fx.Invoke(RegisterWorkerLifecycle),
)

func RegisterConnectionLifecycle(lc fx.Lifecycle, nc *natsLib.Conn) {
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			nc.Close()
			return nil
		},
	})
}

func RegisterSubscriberLifecycle(lc fx.Lifecycle, s *EventSubscriber) {
	lc.Append(fx.Hook{
		OnStart: s.Start,
		OnStop:  s.Stop,
	})
}

func RegisterWorkerLifecycle(lc fx.Lifecycle, w *QueueSubscriber) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error { return w.Start(ctx) },
		OnStop:  func(ctx context.Context) error { return w.Stop(ctx) },
	})
}
