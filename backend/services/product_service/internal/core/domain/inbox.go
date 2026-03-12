package domain

import "time"

// InboxEvent implement Inbox Pattern เพื่อป้องกัน Duplicate Message Processing (Idempotency)
//
// WHY: Kafka อาจส่ง message ซ้ำเมื่อเกิด consumer rebalance หรือ service restart
// ถ้าไม่มีกลไกนี้ เหตุการณ์เช่น DecreaseStock อาจถูกทำซ้ำ ทำให้ stock ติดลบได้
//
// HOW: ก่อน process message ทุกครั้ง Service จะเช็ค ID ใน inbox table ก่อน
// ถ้าเคย process แล้ว → return nil ทันที (skip โดยไม่ error)
// ถ้ายังไม่เคย → process แล้ว save ID ลง table พร้อมกันใน transaction เดิม
type InboxEvent struct {
	// ID ตรงกับ Kafka Message Key ที่ผู้ส่งกำหนด → ใช้เป็น de-duplication key
	ID          string
	// ConsumerID ระบุ service/group ที่ consume เพื่อ scope idempotency ต่อ consumer
	ConsumerID  string
	ProcessedAt time.Time
}
