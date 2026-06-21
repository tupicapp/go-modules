package nats

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FailedMessage is the GORM model for the failed_tasks DLQ table. It records every message that the Worker gave up on
// after MaxDeliver attempts or a terminal error from the handler.
//
// The table name remains "failed_tasks" for backward compatibility with existing migrations. Operators query /
// re-publish from here; nothing inside the running app reads it on a hot path.
type FailedMessage struct {
	ID        uuid.UUID      `gorm:"column:id;type:char(36);primaryKey"`
	Type      string         `gorm:"column:type;type:varchar(255);not null"`
	Version   string         `gorm:"column:version;type:varchar(20);not null"`
	Payload   datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	Attempts  int            `gorm:"column:attempts;type:int;not null"`
	LastError string         `gorm:"column:last_error;type:text;not null"`
	FailedAt  time.Time      `gorm:"column:failed_at;type:timestamp;not null"`
}

func (*FailedMessage) TableName() string { return "failed_tasks" }

// failedMessageRepository persists DLQ rows. Unexported because nothing outside this package should know the DLQ table
// exists.
type failedMessageRepository struct {
	db *gorm.DB
}

func newFailedMessageRepository(db *gorm.DB) *failedMessageRepository {
	return &failedMessageRepository{db: db}
}

// save writes a DLQ row. Uses FirstOrCreate so duplicate terminal-deliveries (e.g. crash between Save and Term) don't
// error out.
func (r *failedMessageRepository) save(ctx context.Context, m *FailedMessage) error {
	res := r.db.WithContext(ctx).
		Where("id = ?", m.ID).
		Attrs(m).
		FirstOrCreate(m)
	return errors.WithStack(res.Error)
}
