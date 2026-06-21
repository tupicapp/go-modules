package inmem_event

import (
	"github.com/tupicapp/go-modules/contract/event"
	"github.com/tupicapp/go-modules/contract/logger"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(newBusFx),
)

// busParams collects the bus dependencies, gathering event middleware from the
// "event_middleware" value group so a service can contribute cross-cutting
// behaviour without re-providing the bus:
//
//	fx.Provide(fx.Annotate(NewTracingMiddleware, fx.ResultTags(`group:"event_middleware"`)))
type busParams struct {
	fx.In
	Logger     logger.Logger
	Middleware []Middleware `group:"event_middleware"`
}

// newBusFx adapts NewBus to fx, spreading the collected middleware group. The
// group is empty by default, so the bus is built with no middleware unless a
// service supplies some.
func newBusFx(p busParams) (event.Publisher, event.Subscriber) {
	return NewBus(p.Logger, p.Middleware...)
}
