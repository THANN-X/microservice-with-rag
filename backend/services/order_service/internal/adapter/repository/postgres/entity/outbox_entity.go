// WHAT: OutboxEventEntity maps to "outbox_event" table
// WHY type:jsonb สำหรับ payload?
//   - PostgreSQL JSONB ทำ index ได้ และ query แบบ JSON path ได้ (เผื่ออนาคต)
//   - ถ้าใช้ MySQL ให้เปลี่ยนเป็น type:text
package entity

import (
	"order_service/internal/core/domain"
	"time"
)

type OutboxEventEntity struct {
	ID            string     `gorm:"primaryKey;type:varchar(36)"` // UUID
	Topic         string     `gorm:"type:varchar(255);not null"`
	AggregateID   string     `gorm:"type:varchar(255);not null;index"` // Kafka message key
	AggregateType string     `gorm:"type:varchar(100);not null"`
	EventType     string     `gorm:"type:varchar(255);not null"`
	Payload       string     `gorm:"type:jsonb;not null"`
	Status        string     `gorm:"type:varchar(50);not null;default:'PENDING';index"` // index for processor query
	ErrorMessage  string     `gorm:"type:text"`
	RetryCount    int        `gorm:"column:retry_count;not null;default:0"`
	SentAt        *time.Time `gorm:"type:timestamptz"`
	CreatedAt     time.Time  `gorm:"autoCreateTime;index"` // index for FIFO ordering
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (OutboxEventEntity) TableName() string { return "outbox_event" }

func (e *OutboxEventEntity) ToOutboxEventDomain() *domain.OutboxEvent {
	if e == nil {
		return nil
	}
	return &domain.OutboxEvent{
		ID:            e.ID,
		Topic:         e.Topic,
		AggregateID:   e.AggregateID,
		AggregateType: e.AggregateType,
		EventType:     e.EventType,
		Payload:       e.Payload,
		Status:        domain.OutboxStatus(e.Status),
		ErrorMessage:  e.ErrorMessage,
		RetryCount:    e.RetryCount,
		SentAt:        e.SentAt,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func ToOutboxEventEntity(d *domain.OutboxEvent) *OutboxEventEntity {
	if d == nil {
		return nil
	}
	return &OutboxEventEntity{
		ID:            d.ID,
		Topic:         d.Topic,
		AggregateID:   d.AggregateID,
		AggregateType: d.AggregateType,
		EventType:     d.EventType,
		Payload:       d.Payload,
		Status:        string(d.Status),
		ErrorMessage:  d.ErrorMessage,
		RetryCount:    d.RetryCount,
		SentAt:        d.SentAt,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
