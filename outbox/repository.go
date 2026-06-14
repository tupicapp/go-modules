package outbox

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/tupicapp/go-modules/persistence/uow"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func newRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) create(ctx context.Context, e *Event) error {
	return errors.WithStack(uow.ORM(ctx, r.db).Create(e).Error)
}

func (r *repository) listUnpublished(ctx context.Context, limit int) ([]*Event, error) {
	if limit <= 0 {
		limit = 100
	}
	var items []*Event
	err := r.db.WithContext(ctx).
		Where("published_at IS NULL AND failed_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return items, nil
}

func (r *repository) markPublished(ctx context.Context, messageID uuid.UUID, publishedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&Event{}).
		Where("message_id = ?", messageID).
		Update("published_at", publishedAt)
	if res.Error != nil {
		return errors.WithStack(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.Newf("outbox: message %s not found", messageID)
	}
	return nil
}

// quarantine permanently excludes an event from future relay attempts by setting failed_at. Only called for permanent
// failures (e.g. marshal errors) where retrying would never succeed regardless of broker state.
func (r *repository) quarantine(ctx context.Context, messageID uuid.UUID, lastError string, failedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&Event{}).
		Where("message_id = ?", messageID).
		Updates(map[string]any{
			"error":     lastError,
			"failed_at": failedAt,
		})
	if res.Error != nil {
		return errors.WithStack(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.Newf("event %s not found for quarantine", messageID)
	}
	return nil
}
