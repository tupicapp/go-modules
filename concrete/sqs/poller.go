package sqs

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/cockroachdb/errors"
	osrouter "github.com/tupicapp/go-modules/concrete/objectstorage_router"
	"github.com/tupicapp/go-modules/concrete/sentry"
	"github.com/tupicapp/go-modules/contract/logger"
	objectstorage2 "github.com/tupicapp/go-modules/contract/objectstorage"
	"github.com/tupicapp/go-modules/kernel/apperror"
	"go.uber.org/zap"
)

const (
	idleBackoff    = 2 * time.Second
	failureBackoff = 5 * time.Second
)

// API is the subset of the AWS SQS client the poller depends on. *sqs.Client
// satisfies it; tests supply a fake so the poll/ack logic is exercised without a
// live queue.
type API interface {
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

type envelope struct {
	Records []record `json:"Records"`
}

type record struct {
	EventName string `json:"eventName"`
	S3        struct {
		Bucket struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key  string `json:"key"`
			Size *int64 `json:"size"`
		} `json:"object"`
	} `json:"s3"`
}

type Poller struct {
	logger logger.Logger
	cfg    Config
	client API
	router *osrouter.Router
	cancel context.CancelFunc
	done   chan struct{}
}

func NewPoller(
	l logger.Logger,
	cfg Config,
	client API,
	router *osrouter.Router,
) (*Poller, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &Poller{
		logger: l,
		cfg:    cfg,
		client: client,
		router: router,
	}, nil
}

func (p *Poller) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.cancel = cancel
	p.done = make(chan struct{})

	p.logger.Info("sqs: poller starting...", zap.String("queue_url", p.cfg.QueueURL))

	go func() {
		defer close(p.done)
		p.run(ctx)
	}()

	return nil
}

func (p *Poller) Stop(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}

	if p.done == nil {
		return nil
	}

	select {
	case <-p.done:
		p.logger.Info("sqs: poller stopped")
		return nil
	case <-ctx.Done():
		return errors.WithStack(ctx.Err())
	}
}

func (p *Poller) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		if err := p.processBatch(ctx); err != nil {
			p.logger.Error("sqs: batch failed", zap.Error(err))

			select {
			case <-ctx.Done():
				return
			case <-time.After(failureBackoff):
			}
		}
	}
}

func (p *Poller) processBatch(ctx context.Context) error {
	output, err := p.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(p.cfg.QueueURL),
		MaxNumberOfMessages: p.cfg.MaxMessages,
		WaitTimeSeconds:     p.cfg.WaitTimeSeconds,
		VisibilityTimeout:   p.cfg.VisibilityTimeout,
	})
	if err != nil {
		return errors.Wrap(err, "sqs: receive messages")
	}

	if len(output.Messages) == 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(idleBackoff):
		}
		return nil
	}

	p.logger.Info("sqs: messages received", zap.Int("message_count", len(output.Messages)))

	for _, msg := range output.Messages {
		p.processMessage(ctx, msg)
	}

	return nil
}

func (p *Poller) processMessage(ctx context.Context, msg types.Message) {
	messageID := aws.ToString(msg.MessageId)
	fields := []zap.Field{zap.String("message_id", messageID)}

	events, err := parseS3Records(aws.ToString(msg.Body))
	if err != nil {
		p.logger.Warn("sqs: parse message failed", append(fields, zap.Error(err))...)
		p.deleteMessage(ctx, msg, fields)
		return
	}

	p.logger.Info("sqs: message parsed", append(fields, zap.Int("record_count", len(events)))...)

	for _, e := range events {
		err = p.router.Handle(e)
		if err == nil {
			continue
		}
		f := append(append([]zap.Field{}, fields...), zap.String("storage_key", e.Key), zap.Error(err))
		if apperror.IsAppError(err) {
			p.logger.Error("sqs: record rejected (terminal)", f...)
			continue
		}
		p.logger.Error("sqs: record processing failed (will retry)", f...)
		sentry.Capture(err, map[string]string{"transport": "sqs", "storage_key": e.Key})
		return
	}

	p.deleteMessage(ctx, msg, fields)
}

func (p *Poller) deleteMessage(ctx context.Context, msg types.Message, fields []zap.Field) {
	if _, err := p.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(p.cfg.QueueURL),
		ReceiptHandle: msg.ReceiptHandle,
	}); err != nil {
		p.logger.Warn("sqs: message delete failed", append(fields, zap.Error(err))...)
		return
	}

	p.logger.Info("sqs: message acknowledged", fields...)
}

func parseS3Records(body string) ([]objectstorage2.Event, error) {
	var env envelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal S3 event")
	}

	events := make([]objectstorage2.Event, 0, len(env.Records))
	for _, r := range env.Records {
		key, err := url.QueryUnescape(r.S3.Object.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot decode S3 key %q", r.S3.Object.Key)
		}
		if key == "" {
			continue
		}
		events = append(events, objectstorage2.Event{
			EventName: r.EventName,
			Bucket:    r.S3.Bucket.Name,
			Key:       key,
			Size:      r.S3.Object.Size,
		})
	}

	if len(events) == 0 {
		return nil, errors.New("sqs: no S3 records in message")
	}

	return events, nil
}

func validateConfig(cfg Config) error {
	if cfg.QueueURL == "" {
		return errors.New("sqs: queue_url is required")
	}
	if cfg.MaxMessages < 1 || cfg.MaxMessages > 10 {
		return errors.New("sqs: max_messages must be between 1 and 10")
	}
	if cfg.WaitTimeSeconds < 0 || cfg.WaitTimeSeconds > 20 {
		return errors.New("sqs: wait_time_seconds must be between 0 and 20")
	}
	if cfg.VisibilityTimeout < 0 {
		return errors.New("sqs: visibility_timeout must be zero or greater")
	}

	return nil
}
