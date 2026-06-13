package outbox

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Event is one row in the outbox_events table. It holds a pending integration event until the Relay publishes it to the
// broker and marks it published.
//
// FailedAt is set only for permanent failures (e.g. marshal errors). Transient broker failures never touch the row —
// the event is simply retried on the next poll tick. Error records the reason for ops inspection and replay.
type Event struct {
	ID          int64          `gorm:"column:id;type:bigserial;primaryKey;autoIncrement"`
	MessageID   uuid.UUID      `gorm:"column:message_id;type:char(36);not null;uniqueIndex"`
	Subject     string         `gorm:"column:subject;type:varchar(255);not null"`
	Version     string         `gorm:"column:version;type:varchar(20);not null"`
	Payload     datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	OccurredAt  time.Time      `gorm:"column:occurred_at;type:timestamp;not null"`
	PublishedAt *time.Time     `gorm:"column:published_at;type:timestamp"`
	Error       *string        `gorm:"column:error;type:text"`
	FailedAt    *time.Time     `gorm:"column:failed_at;type:timestamp"`
}

func (*Event) TableName() string { return "outbox_events" }
