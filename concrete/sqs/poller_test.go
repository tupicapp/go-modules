package sqs

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
	"github.com/tupicapp/go-modules/concrete/noop_logger"
	osrouter "github.com/tupicapp/go-modules/concrete/objectstorage_router"
	objectstorage2 "github.com/tupicapp/go-modules/contract/objectstorage"
	"github.com/tupicapp/go-modules/shared/apperror"
)

// fakeAPI records DeleteMessage calls so ack/keep decisions can be asserted
// without a live queue.
type fakeAPI struct{ deleted int }

func (f *fakeAPI) ReceiveMessage(
	context.Context, *awssqs.ReceiveMessageInput, ...func(*awssqs.Options),
) (*awssqs.ReceiveMessageOutput, error) {
	return &awssqs.ReceiveMessageOutput{}, nil
}

func (f *fakeAPI) DeleteMessage(
	context.Context, *awssqs.DeleteMessageInput, ...func(*awssqs.Options),
) (*awssqs.DeleteMessageOutput, error) {
	f.deleted++
	return &awssqs.DeleteMessageOutput{}, nil
}

func newTestPoller(t *testing.T, api API, r *osrouter.Router) *Poller {
	t.Helper()
	p, err := NewPoller(
		noop_logger.NewNoop(),
		Config{QueueURL: "q", MaxMessages: 1, WaitTimeSeconds: 0, VisibilityTimeout: 0},
		api, r,
	)
	require.NoError(t, err)
	return p
}

func s3Body(eventName, key string) string {
	return fmt.Sprintf(
		`{"Records":[{"eventName":%q,"s3":{"bucket":{"name":"b"},"object":{"key":%q,"size":1}}}]}`,
		eventName, key,
	)
}

func testMessage(body string) types.Message {
	return types.Message{Body: aws.String(body), MessageId: aws.String("m1"), ReceiptHandle: aws.String("r1")}
}

func TestProcessMessage_SuccessDeletes(t *testing.T) {
	api := &fakeAPI{}
	r := osrouter.NewRouter()
	var got objectstorage2.Event
	r.Register("ObjectCreated:*", func(e objectstorage2.Event) error { got = e; return nil })

	newTestPoller(t, api, r).processMessage(context.Background(), testMessage(s3Body("ObjectCreated:Put", "k.jpg")))

	require.Equal(t, 1, api.deleted, "successful handler must delete the message")
	require.Equal(t, "k.jpg", got.Key)
}

func TestProcessMessage_RetriableErrorKeeps(t *testing.T) {
	api := &fakeAPI{}
	r := osrouter.NewRouter()
	r.Register("ObjectCreated:*", func(objectstorage2.Event) error { return errors.New("db unavailable") })

	newTestPoller(t, api, r).processMessage(context.Background(), testMessage(s3Body("ObjectCreated:Put", "k.jpg")))

	require.Equal(t, 0, api.deleted, "retriable error must leave the message visible")
}

func TestProcessMessage_TerminalErrorDeletes(t *testing.T) {
	api := &fakeAPI{}
	r := osrouter.NewRouter()
	r.Register("ObjectCreated:*", func(objectstorage2.Event) error { return apperror.NotFound("asset not found") })

	newTestPoller(t, api, r).processMessage(context.Background(), testMessage(s3Body("ObjectCreated:Put", "k.jpg")))

	require.Equal(t, 1, api.deleted, "terminal (apperror) must delete the message")
}

func TestProcessMessage_MalformedBodyDeletes(t *testing.T) {
	api := &fakeAPI{}
	r := osrouter.NewRouter()

	newTestPoller(t, api, r).processMessage(context.Background(), testMessage("not-json"))

	require.Equal(t, 1, api.deleted, "unparseable message must be deleted, not retried forever")
}
