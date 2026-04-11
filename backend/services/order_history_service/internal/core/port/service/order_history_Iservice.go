package port

import (
	"context"
	"events"
	"order_history_service/internal/core/port/service/dto"
)

// OrderHistoryCommandService — รับ domain events จาก Kafka แล้วอัปเดต read model ใน MongoDB
type OrderHistoryCommandService interface {
	HandleOrderCreated(ctx context.Context, messageID string, evt *events.OrderCreatedEvent) error
	HandleOrderConfirmed(ctx context.Context, messageID string, evt *events.OrderConfirmedEvent) error
	HandleOrderCancelled(ctx context.Context, messageID string, evt *events.OrderCancelledEvent) error
}

// OrderHistoryQueryService — Read Side ให้ user ดูประวัติ order
type OrderHistoryQueryService interface {
	GetOrderByID(ctx context.Context, orderID string, customerID uint) (*dto.OrderHistoryRes, error)
	ListMyOrders(ctx context.Context, customerID uint, req *dto.ListOrderHistoryReq) (*dto.OrderHistoryListRes, error)
}
