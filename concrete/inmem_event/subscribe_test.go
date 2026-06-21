package inmem_event_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tupicapp/go-modules/concrete/inmem_event"
	"github.com/tupicapp/go-modules/concrete/noop_logger"
	"github.com/tupicapp/go-modules/contract/event"
)

type createdEvt struct{ id string }

func (*createdEvt) Name() string { return "test.created" }

type updatedEvt struct{}

func (*updatedEvt) Name() string { return "test.updated" }

func TestOn_DeliversTypedEventToHandler(t *testing.T) {
	pub, sub := inmem_event.NewBus(noop_logger.NewNoop())

	var got *createdEvt
	event.On(sub, func(_ context.Context, e *createdEvt) error {
		got = e
		return nil
	})

	require.NoError(t, pub.Publish(context.Background(), &createdEvt{id: "x"}))
	require.NotNil(t, got)
	require.Equal(t, "x", got.id)
}

func TestOn_IsolatedByEventType(t *testing.T) {
	pub, sub := inmem_event.NewBus(noop_logger.NewNoop())

	called := false
	event.On(sub, func(_ context.Context, _ *updatedEvt) error {
		called = true
		return nil
	})

	require.NoError(t, pub.Publish(context.Background(), &createdEvt{}))
	require.False(t, called, "an updatedEvt handler must not fire for a createdEvt")
}

func TestNewBus_RecoversHandlerPanicAsError(t *testing.T) {
	pub, sub := inmem_event.NewBus(noop_logger.NewNoop())

	event.On(sub, func(_ context.Context, _ *createdEvt) error {
		panic("boom")
	})

	err := pub.Publish(context.Background(), &createdEvt{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "panicked")
}

func TestNewBus_MiddlewareWrapsHandlerOutermostFirst(t *testing.T) {
	var order []string
	tag := func(name string) inmem_event.Middleware {
		return func(next event.Handler) event.Handler {
			return func(ctx context.Context, e event.DomainEvent) error {
				order = append(order, name)
				return next(ctx, e)
			}
		}
	}

	pub, sub := inmem_event.NewBus(noop_logger.NewNoop(), tag("a"), tag("b"))
	event.On(sub, func(_ context.Context, _ *createdEvt) error {
		order = append(order, "handler")
		return nil
	})

	require.NoError(t, pub.Publish(context.Background(), &createdEvt{}))
	require.Equal(t, []string{"a", "b", "handler"}, order)
}
