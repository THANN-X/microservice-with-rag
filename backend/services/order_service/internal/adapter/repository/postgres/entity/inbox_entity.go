// WHAT: InboxEventEntity maps to "inbox_event" table
// Composite PK: (ID, ConsumerID) → same MessageID สามารถ process ได้โดย consumer ที่ต่างกัน
// (ถ้า order_service มี consumer หลายตัวในอนาคต)
package entity

import (
	"order_service/internal/core/domain"
	"time"
)

type InboxEventEntity struct {
	ID          string    `gorm:"primaryKey;type:varchar(255)"`
	ConsumerID  string    `gorm:"primaryKey;type:varchar(255)"` // Composite PK ร่วมกับ ID
	ProcessedAt time.Time `gorm:"autoCreateTime"`
}

func (InboxEventEntity) TableName() string { return "inbox_event" }

func (e *InboxEventEntity) ToInboxEventDomain() *domain.InboxEvent {
	if e == nil {
		return nil
	}
	return &domain.InboxEvent{
		ID:          e.ID,
		ConsumerID:  e.ConsumerID,
		ProcessedAt: e.ProcessedAt,
	}
}

func ToInboxEventEntity(d *domain.InboxEvent) *InboxEventEntity {
	if d == nil {
		return nil
	}
	return &InboxEventEntity{
		ID:         d.ID,
		ConsumerID: d.ConsumerID,
	}
}
