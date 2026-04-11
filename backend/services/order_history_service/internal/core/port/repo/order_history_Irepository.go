package repo

import (
	"context"
	"order_history_service/internal/core/domain"
)

// OrderHistoryWriteRepository — Write Side (อัปเดตจาก Kafka events เท่านั้น)
type OrderHistoryWriteRepository interface {
	Upsert(ctx context.Context, order *domain.OrderHistory) error
	UpdateStatus(ctx context.Context, orderID string, status string) error
	MarkCancelled(ctx context.Context, orderID string, reason string) error
}

// OrderHistoryReadRepository — Read Side (ให้ user ดู order history)
type OrderHistoryReadRepository interface {
	FindByOrderID(ctx context.Context, orderID string) (*domain.OrderHistory, error)
	FindByCustomerID(ctx context.Context, filter domain.OrderHistoryFilter) ([]domain.OrderHistory, int64, error)
}
