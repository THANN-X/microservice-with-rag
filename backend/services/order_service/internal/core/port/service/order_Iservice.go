package port

import (
	"context"
	dto "order_service/internal/core/port/service/dto"
)

// OrderCommandService รับผิดชอบ write use cases ของ Order
type OrderCommandService interface {
	// PlaceOrder สร้าง Order ใหม่และเริ่ม Saga stock reservation
	// Returns OrderRes เพื่อให้ handler ส่ง ID กลับหา client ทันที
	PlaceOrder(ctx context.Context, customerID uint, req *dto.CreateOrderReq) (*dto.OrderRes, error)

	// CancelOrder ยกเลิก Order โดย customer (ownership check ภายใน)
	CancelOrder(ctx context.Context, orderID string, customerID uint, reason string) error

	// AdminCancelOrder ยกเลิก Order โดย admin (ไม่มี ownership check)
	AdminCancelOrder(ctx context.Context, orderID string, reason string) error

	// HandleStockResult ประมวลผล StockReservedEvent จาก product_service (Saga continuation)
	// WHY ไม่รับ events.StockReservedEvent โดยตรง?
	//   - Service layer ไม่ควรรู้จัก Kafka/events package (Hexagonal Architecture)
	//   - ใช้ DTO เป็น "translation layer" ระหว่าง Kafka message และ core service
	HandleStockResult(ctx context.Context, req *dto.HandleStockResultReq) error

	// ProcessPayment ลูกค้าชำระเงิน (CONFIRMED → PAID/AWAITING_PAYMENT)
	ProcessPayment(ctx context.Context, orderID string, customerID uint, req *dto.ProcessPaymentReq) (*dto.PaymentRes, error)

	// HandlePaymentWebhook รับ webhook callback จาก payment gateway
	HandlePaymentWebhook(ctx context.Context, req *dto.PaymentWebhookReq) error
}

// OrderQueryService รับผิดชอบ read use cases ของ Order
// NOTE: ListMyOrders / ListAllOrders ถูกย้ายไป order_history_service (CQRS read side)
//   - order_service เก็บเฉพาะ GetOrderByID สำหรับ sync response หลัง PlaceOrder
//   - รายการย้อนหลังดึงผ่าน BFF → order_history_service (async via Kafka)
type OrderQueryService interface {
	// GetOrderByID ดึง Order เดี่ยวสำหรับ customer (ตรวจ ownership)
	// ใช้สำหรับ sync response ทันทีหลัง PlaceOrder / ProcessPayment
	GetOrderByID(ctx context.Context, orderID string, customerID uint) (*dto.OrderRes, error)
}
