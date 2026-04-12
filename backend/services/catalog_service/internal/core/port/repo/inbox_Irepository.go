package repo

import (
	"catalog_service/internal/core/domain"
	"context"
)

// InboxRepository ใช้เก็บ message ID ที่ consume แล้ว เพื่อป้องกัน double-processing
type InboxRepository interface {
	// HasProcessed ตรวจสอบว่า message นี้ถูก consume ไปแล้วหรือยัง
	HasProcessed(ctx context.Context, messageID, consumerID string) (bool, error)

	// MarkProcessed บันทึกว่า message นี้ถูก process แล้ว
	// ถ้า insert ซ้ำ (duplicate key) ให้ถือว่าสำเร็จ (idempotent)
	MarkProcessed(ctx context.Context, msg *domain.InboxMessage) error
}
