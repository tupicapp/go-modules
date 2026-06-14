package event_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/event"
	"github.com/tupicapp/go-modules/logger"
)

type stubEvent struct{ name string }

func (e stubEvent) Name() string { return e.name }

type EventBusSuite struct{ suite.Suite }

func TestEventBus(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EventBusSuite))
}

func (s *EventBusSuite) newBus() (event.Publisher, event.Subscriber) {
	return event.NewBus(logger.NewNoop())
}

func (s *EventBusSuite) handler(fn func(context.Context, event.DomainEvent) error) event.Handler {
	return fn
}

func (s *EventBusSuite) TestPublish_NoHandlers_ReturnsNil() {
	pub, _ := s.newBus()
	s.NoError(pub.Publish(context.Background(), stubEvent{"no.handlers"}))
}

func (s *EventBusSuite) TestPublish_CallsRegisteredHandler() {
	pub, sub := s.newBus()
	called := false
	sub.Subscribe("user.created", s.handler(func(_ context.Context, _ event.DomainEvent) error {
		called = true
		return nil
	}))

	s.Require().NoError(pub.Publish(context.Background(), stubEvent{"user.created"}))
	s.True(called)
}

func (s *EventBusSuite) TestPublish_CallsAllHandlersInOrder() {
	pub, sub := s.newBus()
	var order []int
	for i := range 3 {
		sub.Subscribe("ordered", s.handler(func(_ context.Context, _ event.DomainEvent) error {
			order = append(order, i)
			return nil
		}))
	}

	s.Require().NoError(pub.Publish(context.Background(), stubEvent{"ordered"}))
	s.Equal([]int{0, 1, 2}, order)
}

func (s *EventBusSuite) TestPublish_HandlerError_AbortsAndReturnsError() {
	pub, sub := s.newBus()
	boom := errors.New("boom")
	secondCalled := false

	sub.Subscribe("failing", s.handler(func(_ context.Context, _ event.DomainEvent) error { return boom }))
	sub.Subscribe("failing", s.handler(func(_ context.Context, _ event.DomainEvent) error {
		secondCalled = true
		return nil
	}))

	err := pub.Publish(context.Background(), stubEvent{"failing"})
	s.ErrorContains(err, "boom")
	s.False(secondCalled, "second handler must not be called after the first fails")
}

func (s *EventBusSuite) TestPublish_PassesEventToHandler() {
	pub, sub := s.newBus()
	var received event.DomainEvent
	sub.Subscribe("typed", s.handler(func(_ context.Context, e event.DomainEvent) error {
		received = e
		return nil
	}))

	s.Require().NoError(pub.Publish(context.Background(), stubEvent{"typed"}))
	s.Equal("typed", received.Name())
}

func (s *EventBusSuite) TestPublish_HandlersIsolatedByEventName() {
	pub, sub := s.newBus()
	var called []string
	sub.Subscribe("a", s.handler(func(_ context.Context, e event.DomainEvent) error {
		called = append(called, e.Name())
		return nil
	}))
	sub.Subscribe("b", s.handler(func(_ context.Context, e event.DomainEvent) error {
		called = append(called, e.Name())
		return nil
	}))

	s.Require().NoError(pub.Publish(context.Background(), stubEvent{"a"}))
	s.Equal([]string{"a"}, called)
}

func (s *EventBusSuite) TestPublish_ConcurrentPublishIsSafe() {
	pub, sub := s.newBus()
	var mu sync.Mutex
	count := 0
	sub.Subscribe("concurrent", s.handler(func(_ context.Context, _ event.DomainEvent) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	}))

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pub.Publish(context.Background(), stubEvent{"concurrent"})
		}()
	}
	wg.Wait()
	s.Equal(50, count)
}

func (s *EventBusSuite) TestPublishAll_PublishesEachEventInOrder() {
	pub, sub := s.newBus()
	var got []string
	for _, name := range []string{"a", "b"} {
		sub.Subscribe(name, s.handler(func(_ context.Context, e event.DomainEvent) error {
			got = append(got, e.Name())
			return nil
		}))
	}

	err := pub.PublishAll(context.Background(), []event.DomainEvent{stubEvent{"a"}, stubEvent{"b"}})
	s.Require().NoError(err)
	s.Equal([]string{"a", "b"}, got)
}

func (s *EventBusSuite) TestPublishAll_AbortsOnFirstError() {
	pub, sub := s.newBus()
	boom := errors.New("boom")
	secondHandled := false
	sub.Subscribe("first", s.handler(func(_ context.Context, _ event.DomainEvent) error { return boom }))
	sub.Subscribe("second", s.handler(func(_ context.Context, _ event.DomainEvent) error {
		secondHandled = true
		return nil
	}))

	err := pub.PublishAll(context.Background(), []event.DomainEvent{stubEvent{"first"}, stubEvent{"second"}})
	s.ErrorContains(err, "boom")
	s.False(secondHandled, "events after the failing one must not be published")
}

func (s *EventBusSuite) TestPublishAll_EmptyReturnsNil() {
	pub, _ := s.newBus()
	s.NoError(pub.PublishAll(context.Background(), nil))
}
