// WHAT: PaymentRepository interface — กำหนด contract สำหรับ persistence ของ Payment entity
//
// WHY ต้องมี FindByOrderID และ FindByGatewayChargeID แยกกัน?
//   - FindByOrderID    — ใช้ใน ProcessPayment (idempotency check) และ refundIfPaid (cancel flow)
//   - FindByGatewayChargeID — ใช้ใน HandlePaymentWebhook เพราะ gateway ส่ง chargeID มา ไม่ใช่ orderID
//     (เราไม่มีทางรู้ว่า chargeID XXX เป็นของ order ไหน โดยไม่ query ผ่าน chargeID)
package port

import (
	"context"
	"order_service/internal/core/domain"
)

type PaymentRepository interface {
	// Create บันทึก Payment record ใหม่ (สถานะ PENDING)
	Create(ctx context.Context, payment *domain.Payment) error

	// FindByOrderID ดึง payment ล่าสุดของ order นั้น (ถ้ามีหลาย record คืน created_at ใหม่สุด)
	FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)

	// FindByGatewayChargeID ดึง payment ด้วย chargeID ที่ gateway ออกให้
	// WHY? — webhook callback จาก gateway มีแค่ chargeID, ไม่มี orderID
	FindByGatewayChargeID(ctx context.Context, chargeID string) (*domain.Payment, error)

	// UpdateStatus อัปเดตเฉพาะ fields ที่เปลี่ยนตาม status (status, charge_id, paid_at, failed_reason)
	UpdateStatus(ctx context.Context, payment *domain.Payment) error
}
