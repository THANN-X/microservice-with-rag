// WHAT: Payment entity — บันทึกการชำระเงินแต่ละครั้ง แยกออกจาก Order aggregate
//
// WHY แยก Payment ออกจาก Order?
//   - Audit trail: ถ้า customer จ่ายหลายครั้ง (retry / refund) ต้องมีหลาย Payment record ต่อ 1 Order
//   - Single Responsibility: Order รู้ว่า "จ่ายแล้วหรือยัง" แต่ Payment รู้ว่า "จ่ายผ่านช่องทางไหน"
//   - GatewayChargeID คือ ID ของ gateway (Stripe/Omise) ที่ใช้ refund/track ทีหลัง
package domain

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "PENDING"  // สร้าง record แล้ว แต่ยังไม่ได้ส่งไป gateway
	PaymentStatusSuccess  PaymentStatus = "SUCCESS"  // gateway ยืนยันว่าจ่ายสำเร็จ
	PaymentStatusFailed   PaymentStatus = "FAILED"   // gateway ปฏิเสธ หรือ timeout
	PaymentStatusRefunded PaymentStatus = "REFUNDED" // คืนเงินสำเร็จแล้ว
)

type Payment struct {
	ID              string
	OrderID         string
	CustomerID      uint
	Amount          float64
	Currency        string
	Status          PaymentStatus
	Gateway         string     // ชื่อ gateway ที่ใช้ เช่น "STRIPE", "OMISE", "STUB"
	GatewayChargeID string     // ID ของ charge ที่ gateway ออกให้ — ใช้ refund/reconcile กับ gateway
	PaymentMethod   string     // เช่น "CREDIT_CARD", "PROMPTPAY"
	PaidAt          *time.Time // WHY pointer? — nil ถ้ายังไม่จ่าย, มีค่าเมื่อ SUCCESS เท่านั้น
	FailedReason    string     // เก็บเหตุผลที่ gateway ปฏิเสธ เช่น "insufficient funds"
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewPayment(orderID string, customerID uint, amount float64, currency, gateway, method string) *Payment {
	return &Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		CustomerID:    customerID,
		Amount:        amount,
		Currency:      currency,
		Status:        PaymentStatusPending, // เริ่มต้นเสมอที่ PENDING ก่อน call gateway
		Gateway:       gateway,
		PaymentMethod: method,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// MarkSuccess เรียกหลัง gateway ยืนยัน charge สำเร็จ
// WHY ไม่ return error? — state machine สั้น ไม่มี invariant ที่ซับซ้อน
//
//	ถ้า business logic ซับซ้อนขึ้น (เช่น ต้องเช็คสถานะเดิม) ค่อยเปลี่ยนเป็น error return
func (p *Payment) MarkSuccess(chargeID string) {
	now := time.Now()
	p.Status = PaymentStatusSuccess
	p.GatewayChargeID = chargeID // บันทึก charge ID สำหรับ refund ในอนาคต
	p.PaidAt = &now
	p.UpdatedAt = now
}

func (p *Payment) MarkFailed(reason string) {
	p.Status = PaymentStatusFailed
	p.FailedReason = reason
	p.UpdatedAt = time.Now()
}

func (p *Payment) MarkRefunded() {
	p.Status = PaymentStatusRefunded
	// WHY ไม่ล้าง GatewayChargeID? — ยังต้องใช้ ID นี้เพื่อ audit / reconcile กับ gateway
	p.UpdatedAt = time.Now()
}
