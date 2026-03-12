package domain

import (
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

// สถานะของ Outbox Message ใน Lifecycle
//  PENDING → message รอ Outbox Processor มาหยิบส่ง Kafka
//  SENT    → ส่ง Kafka สำเร็จแล้ว (ปลอดภัยแล้ว ไม่ต้องส่งซ้ำ)
//  FAILED  → ส่งเกิน maxRetries แล้ว → ถือว่า Dead Letter (ต้องมี alert หรือ manual retry)
const (
	OutboxStatusPending OutboxStatus = "PENDING"
	OutboxStatusSent    OutboxStatus = "SENT"
	OutboxStatusFailed  OutboxStatus = "FAILD" // TODO: แก้ typo → "FAILED"
)

// OutboxMessage คือ Entity หลักที่เราจะใช้ใน Business Logic
type OutboxEvent struct {
	ID            string       // UUID
	Topic         string       // ชื่อ Topic ที่จะส่งไป (ถ้ามี)
	AggregateID   string       // ID ของสินค้า หรือ Order (เพื่อใช้ทำ Partition Key ได้)
	AggregateType string       // เช่น "PRODUCT", "ORDER"
	EventType     string       // ชื่อ Event เช่น "StockDecreased"
	Payload       string       // JSON String ข้อมูลจริงๆ
	Status        OutboxStatus // สถานะ
	ErrorMessage  string       // เก็บ error ถ้าส่งไม่ผ่าน (เผื่อไว้ debug)
	RetryCount    int          // นับจำนวนครั้งที่ลองส่ง
	SentAt        *time.Time   // เวลาที่ส่งสำเร็จ (Pointer เพื่อให้เป็น null ได้)
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Helper Function สำหรับสร้าง Message ใหม่
func NewOutboxMessage(topic, aggregateID, aggregateType, eventType, payload string) *OutboxEvent {
	return &OutboxEvent{
		/* ID: ปกติจะ Gen UUID ที่นี่ หรือที่ Repo ก็ได้ แต่ Clean Arch มัก Gen ที่นี่
		สมมติว่าคุณมี lib uuid: uuid.New().String() */

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

func (o *OutboxEvent) UpdatePayload(payload string) {
	o.Payload = payload
	o.UpdatedAt = time.Now()
}

/*Pure Object
func (o *OutboxEvent) MarkAsSent() {
	o.Status = OutboxStatusSent
	o.UpdatedAt = time.Now()
} */

/*func (o *OutboxEvent) MarkAsFailed(errMsg string) {
	o.Status = OutboxStatusFailed
	o.ErrorMessage = errMsg
	o.UpdatedAt = time.Now()
} */
