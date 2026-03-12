package entity

import (
	"product_service/internal/core/domain"
	"time"
)

type InboxEventEntity struct {
	ID          string    `gorm:"primaryKey;varchar(255)"`
	ConsumerID  string    `gorm:"type:varchar(255);primaryKey"`
	ProcessedAt time.Time `gorm:"autoCreateTime"`
}

func (InboxEventEntity) TableName() string {
	return "inbox_event"
}

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
		ID:          d.ID,
		ConsumerID:  d.ConsumerID,
		ProcessedAt: d.ProcessedAt,
	}
}
