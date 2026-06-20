// WHAT: OrderCommandRepository กำหนด write-side contract สำหรับ Order aggregate
// WHY แยก Command และ Query interface?
//   - Command side ต้องการ TX + domain event support
//   - Query side เน้น read (อาจย้ายไป read replica ในอนาคต)
//   - เปลี่ยน implementation ฝั่งใดฝั่งหนึ่งโดยไม่กระทบอีกฝั่ง (Interface Segregation)
package port

import (
	"context"
	"order_service/internal/core/domain"
	"time"
)

// OrderCommandRepository รับผิดชอบ write operations ของ Order aggregate
type OrderCommandRepository interface {
	TransactionManager

	// CreateOrder INSERT Order + OrderItems ทั้งหมดในครั้งเดียว
	// WHY รับ *domain.Order แทนแยก fields?
	//   - Aggregate root กำหนด invariants → ส่ง whole aggregate ให้ repo handle mapping
	//   - ง่ายต่อการ add fields ในอนาคตโดยไม่เปลี่ยน interface
	CreateOrder(ctx context.Context, order *domain.Order) error

	// GetOrderByID โหลด Order พร้อม OrderItems สำหรับ command operations
	// WHY ต้อง preload items?
	//   - Cancel/RequestStockRelease ต้องการ items เพื่อสร้าง OrderCancelledEvent
	//   - Load แบบ lazy อาจทำให้ domain logic ทำงานกับข้อมูลไม่ครบ
	GetOrderByID(ctx context.Context, id string) (*domain.Order, error)

	// UpdateOrderStatus ทำ targeted UPDATE เฉพาะ status field
	// WHY ไม่ใช้ full Save?
	//   - ป้องกัน race condition กับ concurrent updates (e.g. timeout worker + user cancel)
	//   - SQL targeted update: UPDATE orders SET status=? WHERE id=?
	//   - TODO: เพิ่ม optimistic lock (version field) ถ้าต้องการ strict concurrency control
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error

	// SaveDomainEvents บันทึก domain events จาก Order aggregate ลง outbox table
	// WHY อยู่ใน OrderCommandRepository แทน OutboxRepository?
	//   - เพื่อให้ Save ใน TX context เดียวกับ CreateOrder/UpdateOrderStatus
	//   - Transactional Outbox: event + state change atomic เสมอ
	//   - ถ้า TX fail → ทั้ง order state และ outbox event หายไปพร้อมกัน (ไม่มี inconsistency)
	SaveDomainEvents(ctx context.Context, order *domain.Order) error

	// FindExpiredPaymentOrders ค้นหา orders ที่อยู่ใน AWAITING_PAYMENT นานเกินกำหนด
	FindExpiredPaymentOrders(ctx context.Context, timeout time.Duration) ([]*domain.Order, error)
}

// OrderQueryRepository รับผิดชอบ read operations (ไม่ต้องการ TX หรือ event)
// NOTE: FindByCustomerID / FindAll ถูกย้ายไป order_history_service (CQRS read side)
type OrderQueryRepository interface {
	// FindByID ดึง Order เดี่ยวสำหรับ sync response หลัง PlaceOrder (รวม items)
	FindByID(ctx context.Context, id string) (*domain.Order, error)
}
