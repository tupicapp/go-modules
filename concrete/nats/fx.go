package nats

import (
	"context"

	natsLib "github.com/nats-io/nats.go"
	msgrouter "github.com/tupicapp/go-modules/concrete/messaging_router"
	messaging2 "github.com/tupicapp/go-modules/contract/messaging"
	"go.uber.org/fx"
)

// ConnectionModule provides the NATS connection and JetStreamContext. Requires a nats.Config in the graph, supplied by
// the service.
var ConnectionModule = fx.Options(
	fx.Provide(NewConnection),
	fx.Invoke(RegisterConnectionLifecycle),
)

// SubscriberModule wires the shared Router, the EventSubscriber (non-queues.* subjects), and exposes the Router as a
// MessageHandlerRegisterer.
var SubscriberModule = fx.Options(
	fx.Provide(
		msgrouter.NewRouter,
		NewEventSubscriber,
		func(r *msgrouter.Router) messaging2.MessageHandlerRegisterer { return r },
	),
	fx.Invoke(RegisterSubscriberLifecycle),
)

// WorkerModule wires the QueueSubscriber (queues.* subjects). Requires the shared Router already provided by
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
