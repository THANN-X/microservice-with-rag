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

func (e *InboxEventEntity) ToIndboxEventDomain() *domain.InboxEvent {
	return &domain.InboxEvent{
		ID:          e.ID,
		ConsumerID:  e.ConsumerID,
		ProcessedAt: e.ProcessedAt,
	}
}

func ToIndboxEventEntity(d *domain.InboxEvent) *InboxEventEntity {
	return &InboxEventEntity{
		ID:          d.ID,
		ConsumerID:  d.ConsumerID,
		ProcessedAt: d.ProcessedAt,
	}
}
