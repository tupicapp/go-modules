package sqs

import (
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	osrouter "github.com/tupicapp/go-modules/concrete/objectstorage_router"
	objectstorage2 "github.com/tupicapp/go-modules/contract/objectstorage"
	"go.uber.org/fx"
)

// Module wires the object-storage Router, the SQS client, and the Poller
// lifecycle. It provides the Router as an objectstorage.EventHandlerRegisterer so
// services register handlers without depending on this adapter. Requires a
// sqs.Config in the graph, supplied by the service (mirrors nats.ConnectionModule).
var Module = fx.Options(
	fx.Provide(
		osrouter.NewRouter,
		func(r *osrouter.Router) objectstorage2.EventHandlerRegisterer { return r },
		NewClient,
		func(c *awssqs.Client) API { return c },
		NewPoller,
	),
	fx.Invoke(RegisterLifecycle),
)

func RegisterLifecycle(lc fx.Lifecycle, p *Poller) {
	lc.Append(fx.Hook{
		OnStart: p.Start,
		OnStop:  p.Stop,
	})
}
