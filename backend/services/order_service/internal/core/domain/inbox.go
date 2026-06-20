// WHAT: InboxEvent domain object — บันทึก message IDs ที่ประมวลผลไปแล้ว
// WHY: Inbox Pattern ทำให้ Kafka Consumer เป็น Idempotent
//   - Kafka guarantees at-least-once delivery → ข้อความอาจถูกส่งซ้ำ (consumer rebalance, crash)
//   - ถ้าไม่มี Inbox → HandleStockResult ถูกเรียกซ้ำ → order ถูก confirm/cancel ซ้ำ
//   - ตรวจสอบ MessageID ก่อนประมวลผล ถ้าเคย process แล้ว → skip (idempotent)
package domain

import "time"

type InboxEvent struct {
	ID          string    // MessageID จาก Kafka message (ใช้ AggregateID ที่ producer ตั้งไว้)
	ConsumerID  string    // ชื่อ consumer group สำหรับ scope idempotency ต่อ consumer
	ProcessedAt time.Time
}
