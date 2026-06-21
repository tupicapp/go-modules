// Package sqs is the SQS adapter for object-storage events: it polls an SQS queue
// fed by S3 bucket notifications, parses each message into an objectstorage.Event,
// and dispatches it through an objectstorage.Router. Application handlers depend
// on objectstorage, not on this package.
package sqs

// Config carries the SQS connection and polling parameters. The mapstructure tags let a
// service load it directly from JSON (alias it) instead of redeclaring the schema.
type Config struct {
	QueueURL          string `mapstructure:"queue_url"`
	AWSRegion         string `mapstructure:"aws_region"`
	Key               string `mapstructure:"key"` // optional static credential; falls back to the default AWS chain (IRSA) when empty
	Secret            string `mapstructure:"secret"`
	Endpoint          string `mapstructure:"endpoint"` // optional custom endpoint (e.g. LocalStack)
	MaxMessages       int32  `mapstructure:"max_messages"`
	WaitTimeSeconds   int32  `mapstructure:"wait_time_seconds"`
	VisibilityTimeout int32  `mapstructure:"visibility_timeout"`
}
