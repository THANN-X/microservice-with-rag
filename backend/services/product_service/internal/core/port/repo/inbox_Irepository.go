package port

import (
	"context"
	"product_service/internal/core/domain"
)

type InboxRepository interface {
	// Idempotency (Inbox Pattern)

	// ใช้ตรวจสอบว่า Message ID นี้เคยทำไปหรือยัง
	HasProcessedMessage(ctx context.Context, messageID string) (bool, error)
	// บันทึกว่าทำเสร็จแล้ว
	SaveProcessedMessage(ctx context.Context, event *domain.InboxEvent) error
}
