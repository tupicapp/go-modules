package outbox_queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/outbox_queue"
	"github.com/tupicapp/go-modules/contract/outbox"
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

// stubTask is a minimal queue.Task. Data is exported so the marshaled payload
// can be asserted.
type stubTask struct {
	name    string
	version string
	Data    string `json:"data"`
}

func (t stubTask) Name() string    { return t.name }
func (t stubTask) Version() string { return t.version }

type QueueSuite struct{ suite.Suite }

func TestQueueSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(QueueSuite))
}

func (s *QueueSuite) TestEnqueue_UsesDefaultChannel() {
	ob := &stubOutbox{}
	q := outbox_queue.NewOutboxQueue(ob)

	err := q.Enqueue(context.Background(), stubTask{name: "validate-asset", version: "2"})
	s.Require().NoError(err)

	s.Require().NotNil(ob.stored)
	s.Equal("queues.default.validate-asset", ob.stored.Subject())
	s.Equal("2", ob.stored.Version())
}

func (s *QueueSuite) TestEnqueueOn_UsesGivenChannel() {
	ob := &stubOutbox{}
	q := outbox_queue.NewOutboxQueue(ob)

	err := q.EnqueueOn(context.Background(), "high", stubTask{name: "validate-asset", version: "1"})
	s.Require().NoError(err)
	s.Equal("queues.high.validate-asset", ob.stored.Subject())
}

// TestEnqueue_MarshalsTaskPayloadAtTopLevel guards against re-wrapping the task
// in a {"task":...} envelope — the stored payload must be the task's own JSON.
func (s *QueueSuite) TestEnqueue_MarshalsTaskPayloadAtTopLevel() {
	ob := &stubOutbox{}
	q := outbox_queue.NewOutboxQueue(ob)

	err := q.Enqueue(context.Background(), stubTask{name: "x", version: "1", Data: "hello"})
	s.Require().NoError(err)

	payload, err := json.Marshal(ob.stored)
	s.Require().NoError(err)
	s.JSONEq(`{"data":"hello"}`, string(payload))
}

func (s *QueueSuite) TestEnqueue_PropagatesOutboxError() {
	boom := errors.New("db down")
	q := outbox_queue.NewOutboxQueue(&stubOutbox{err: boom})

	err := q.Enqueue(context.Background(), stubTask{name: "x", version: "1"})
	s.Require().Error(err)
	s.ErrorIs(err, boom)
}
