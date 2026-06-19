package queue_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/outbox"
	"github.com/tupicapp/go-modules/queue"
)

// stubOutbox records the last event handed to Store and can be primed to fail.
type stubOutbox struct {
	stored outbox.IntegrationEvent
	err    error
}

func (o *stubOutbox) Store(_ context.Context, e outbox.IntegrationEvent) error {
	o.stored = e
	return o.err
}

// stubTask is a minimal queue.Task.
type stubTask struct {
	subject string
	version string
}

func (t stubTask) Subject() string { return t.subject }
func (t stubTask) Version() string { return t.version }

type QueueSuite struct{ suite.Suite }

func TestQueueSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(QueueSuite))
}

func (s *QueueSuite) TestEnqueue_StoresTaskAsEvent() {
	ob := &stubOutbox{}
	q := queue.NewOutboxQueue(ob)

	err := q.Enqueue(context.Background(), stubTask{subject: "queue.validate-asset", version: "2"})
	s.Require().NoError(err)

	// The task must reach the outbox preserving its subject/version shape.
	s.Require().NotNil(ob.stored)
	s.Equal("queue.validate-asset", ob.stored.Subject())
	s.Equal("2", ob.stored.Version())
}

func (s *QueueSuite) TestEnqueue_PropagatesOutboxError() {
	boom := errors.New("db down")
	q := queue.NewOutboxQueue(&stubOutbox{err: boom})

	err := q.Enqueue(context.Background(), stubTask{subject: "queue.x", version: "1"})
	s.Require().Error(err)
	s.ErrorIs(err, boom)
}
