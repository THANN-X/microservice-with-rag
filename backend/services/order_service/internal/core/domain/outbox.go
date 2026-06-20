// WHAT: OutboxEvent domain object — แทน "pending message" ที่รอส่งไป Kafka
// WHY: ใช้ Transactional Outbox Pattern เพื่อการันตี At-Least-Once delivery
//   - บันทึก event ลง DB ใน TX เดียวกับ business data
//   - OutboxProcessor (background worker) อ่านแล้วส่งไป Kafka
//   - ถ้า service crash ก่อนส่ง Kafka → restart แล้ว processor ส่งใหม่ได้
// TODO: ถ้าต้องการ latency ต่ำ พิจารณา CDC (Change Data Capture) แทน polling
package domain

import (
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	OutboxStatusPending OutboxStatus = "PENDING"
	OutboxStatusSent    OutboxStatus = "SENT"
	OutboxStatusFailed  OutboxStatus = "FAILED" // Dead Letter หลังจาก retry เกิน maxRetries
)

type OutboxEvent struct {
	ID            string
	Topic         string
	AggregateID   string // Kafka message key → ordering ต่อ aggregate
	AggregateType string
	EventType     string
	Payload       string       // JSON serialized event
	Status        OutboxStatus
	ErrorMessage  string
	RetryCount    int
	SentAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewOutboxMessage สร้าง OutboxEvent ใหม่พร้อม UUID และ PENDING status
func NewOutboxMessage(topic, aggregateID, aggregateType, eventType, payload string) *OutboxEvent {
	return &OutboxEvent{
		ID:            uuid.New().String(),
		Topic:         topic,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		Payload:       payload,
		Status:        OutboxStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}
