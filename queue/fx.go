package queue

import "go.uber.org/fx"

// Module provides the outbox-backed queue as the Queue contract.
var Module = fx.Options(
	fx.Provide(fx.Annotate(NewOutboxQueue, fx.As(new(Queue)))),
)
