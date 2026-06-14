package nats

import (
	"github.com/cockroachdb/errors"
	natsLib "github.com/nats-io/nats.go"
	"github.com/tupicapp/go-modules/logger"
	"go.uber.org/zap"
)

func NewConnection(l logger.Logger, cfg Config) (*natsLib.Conn, natsLib.JetStreamContext, error) {
	if cfg.URL == "" {
		return nil, nil, errors.New("nats.url is required")
	}

	opts := []natsLib.Option{natsLib.Name(cfg.AppSlug)}
	if cfg.Token != "" {
		opts = append(opts, natsLib.Token(cfg.Token))
	}

	nc, err := natsLib.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot connect to nats")
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, errors.Wrap(err, "cannot create NATS JetStream context")
	}

	l.Info("nats: connected", zap.String("url", cfg.URL))
	return nc, js, nil
}
