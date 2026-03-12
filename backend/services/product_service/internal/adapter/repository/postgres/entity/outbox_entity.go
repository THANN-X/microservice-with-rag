package entity

import (
	"product_service/internal/core/domain"
	"time"
)

type OutboxEventEntity struct {
	ID            string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"` // หรือ varchar ถ้าไม่มี extension
	Topic         string     `gorm:"type:varchar(255);"`                             // ชื่อ Topic ที่จะส่งไป (ถ้ามี)
	AggregateID   string     `gorm:"type:varchar(255);not null;index"`               // index ไว้หาตาม ID ง่ายๆ
	AggregateType string     `gorm:"type:varchar(100);not null"`
	EventType     string     `gorm:"type:varchar(255);not null"`
	Payload       string     `gorm:"type:jsonb;not null"` // ใช้ jsonb ถ้าเป็น Postgres จะดีมาก
	Status        string     `gorm:"type:varchar(50);not null;default:'PENDING'"`
	ErrorMessage  string     `gorm:"type:text"`
	RetryCount    int        `gorm:"column:retry_count;not null;default:0"`
	SentAt        *time.Time `gorm:"type:timestamptz"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (OutboxEventEntity) TableName() string {
	return "outbox_event"
}

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
		Status:        domain.OutboxStatus(e.Status), // Cast string กลับเป็น custom type
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
		Status:        string(d.Status), // Cast custom type กลับเป็น string
		ErrorMessage:  d.ErrorMessage,
		RetryCount:    d.RetryCount,
		SentAt:        d.SentAt,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
