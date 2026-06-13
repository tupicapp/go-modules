package outbox

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupicapp/common-go/clock"
	"gorm.io/gorm"
)

// Storage implements the Outbox port by writing each event to the outbox_events table in the caller's transaction
// (Transactional Outbox). The Relay picks up unpublished rows and ships them to the central event bus.
type Storage struct {
	clock      clock.Clock
	repository *repository
}

func NewStorage(c clock.Clock, db *gorm.DB) *Storage {
	return &Storage{clock: c, repository: newRepository(db)}
}

func (p *Storage) Store(ctx context.Context, e OutboxEvent) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return errors.Wrapf(err, "outbox storage: marshal %q", e.Subject())
	}

	id, err := uuid.NewV7()
	if err != nil {
		return errors.WithStack(err)
	}

	if err = p.repository.create(ctx, &Event{
		MessageID:  id,
		Subject:    e.Subject(),
		Version:    e.Version(),
		Payload:    payload,
		OccurredAt: p.clock.Now().UTC(),
	}); err != nil {
		return errors.Wrapf(err, "cannot persist %q %q", e.Subject(), e.Version())
	}
	return nil
}

var _ Outbox = (*Storage)(nil)
