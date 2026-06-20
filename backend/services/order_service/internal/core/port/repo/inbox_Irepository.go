// WHAT: InboxRepository กำหนด contract สำหรับ Idempotent Consumer pattern
// WHY Inbox Pattern?
//   - Kafka at-least-once → StockReservedEvent อาจถูกส่งซ้ำเมื่อ consumer crash ระหว่าง process
//   - ถ้าไม่มี inbox check → HandleStockResult อาจ confirm/cancel order ซ้ำ
//   - Inbox table acts as "processed message log" → check before process, save after
package port

import (
	"context"
	"order_service/internal/core/domain"
)

type InboxRepository interface {
	// HasProcessedMessage ตรวจสอบว่า MessageID นี้เคยถูก process สำเร็จหรือยัง
	HasProcessedMessage(ctx context.Context, messageID string) (bool, error)

	// SaveProcessedMessage บันทึก MessageID หลังจาก process สำเร็จ
	// ต้องเรียกภายใน TX เดียวกับ business operation เพื่อ atomic exactly-once semantics
	SaveProcessedMessage(ctx context.Context, event *domain.InboxEvent) error
}
