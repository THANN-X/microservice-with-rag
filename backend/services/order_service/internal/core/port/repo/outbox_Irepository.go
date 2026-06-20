// WHAT: OutboxRepository กำหนด contract สำหรับ background worker (OutboxProcessor)
// WHY แยกออกจาก OrderCommandRepository?
//   - OutboxProcessor ต้องการ read/update outbox สำหรับ batch processing
//   - ไม่เกี่ยวกับ Order aggregate → แยก interface ทำให้ dependency ชัดเจน
package port

import (
	"context"
	"order_service/internal/core/domain"
)

type OutboxRepository interface {
	// Save บันทึก outbox event ใหม่ (ใช้ใน TX context จาก SaveDomainEvents)
	Save(ctx context.Context, event *domain.OutboxEvent) error

	// GetUnsentMessages ดึง PENDING events เรียง FIFO (LIMIT batch)
	// WHY FIFO? — รับประกัน ordering: event เก่าถูกส่งก่อนเสมอ (e.g. Created ก่อน Cancelled)
	GetUnsentMessages(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)

	// MarkAsSent อัปเดต status=SENT + sent_at เมื่อ Kafka ตอบ ack
	MarkAsSent(ctx context.Context, id string) error

	// IncrementRetryCount เพิ่ม retry_count แบบ atomic (SQL: retry_count = retry_count + 1)
	// WHY atomic? — ป้องกัน lost update ถ้า processor รัน 2+ instances
	IncrementRetryCount(ctx context.Context, id string) error

	// MarkAsFailed ตั้ง status=FAILED + error_message (Dead Letter)
	// เรียกเมื่อ retry_count เกิน maxRetries
	MarkAsFailed(ctx context.Context, id, errMsg string) error
}
