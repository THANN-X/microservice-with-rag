// WHAT: PaymentGateway interface — กำหนด contract ที่ payment logic ใช้
//
// WHY ใช้ interface แทนเรียก Stripe/Omise ตรงๆ?
//   - ทดสอบได้: unit test ใช้ StubPaymentGateway, ไม่ต้องมี API key หรือ internet
//   - Swap ได้: เปลี่ยนจาก Omise เป็น Stripe → แก้แค่ implementation ไม่แตะ business logic
//   - Clean Architecture: core domain ไม่รู้จัก "Stripe" — รู้จักแค่ PaymentGateway
//
// Implementations:
//   - StripeGateway  → adapter/client/stripe_payment_gateway.go  (credit card + PromptPay QR)
//   - StubPaymentGateway → adapter/client/payment_gateway_c.go  (dev/test, always SUCCESS)
package gateway

import "context"

// ChargeRequest ข้อมูลที่ส่งไปให้ gateway เพื่อ charge เงิน
type ChargeRequest struct {
	OrderID    string
	CustomerID uint
	Amount     float64
	Currency   string
	// Token คือ tokenized card จาก frontend (Stripe.js / Omise.js)
	// WHY ใช้ token แทนเลขบัตร?
	//   - PCI-DSS compliance: server ไม่รับเลขบัตรโดยตรง → ลด scope ของ audit
	//   - frontend ส่งเลขบัตรไปที่ gateway โดยตรง → gateway คืน one-time token มาให้ backend
	Token  string
	Method string // เช่น "CREDIT_CARD", "PROMPTPAY"
}

// ChargeResponse ผลลัพธ์จาก gateway หลัง charge
type ChargeResponse struct {
	ChargeID string
	// Status มี 3 ค่า:
	//   "SUCCESS" — จ่ายสำเร็จทันที (บัตรเครดิต)
	//   "PENDING" — รอลูกค้าชำระ (PromptPay / bank transfer) → รอ webhook callback
	//   "FAILED"  — gateway ปฏิเสธ (เงินไม่พอ, บัตรหมดอายุ)
	Status string
	// ClientSecret คือ Stripe PaymentIntent client_secret
	// WHY ต้องส่งไป frontend?
	//   - 3DS: frontend เรียก stripe.confirmCardPayment(clientSecret) เพื่อแสดง 3DS modal
	//   - Credit card: ไม่ต้องใช้ clientSecret ถ้า succeed ทันที
	ClientSecret string
	// QRImageURL คือ URL รูป QR Code สำหรับ PromptPay
	// WHY backend ดึงแทน frontend?
	//   - PromptPay ไม่ต้องการ 3DS/user-input → confirm ที่ backend ได้เลย
	//   - frontend แค่แสดงรูป ไม่ต้องรู้จัก Stripe.js PromptPay API
	QRImageURL string
}

// WebhookEvent ข้อมูลที่ gateway ส่งมาใน callback (async payment เช่น PromptPay)
type WebhookEvent struct {
	ChargeID string `json:"charge_id"`
	OrderID  string `json:"order_id"`
	Status   string `json:"status"`
}

type PaymentGateway interface {
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)

	// VerifyWebhook ตรวจสอบ signature ก่อน parse payload
	// WHY ต้อง verify signature?
	//   - ป้องกัน attacker ที่ส่ง fake webhook มาบอกว่า "order XXX จ่ายแล้ว"
	//   - gateway จะ sign payload ด้วย HMAC/RSA → เราต้อง verify ด้วย secret key
	VerifyWebhook(signature string, payload []byte) (*WebhookEvent, error)

	Refund(ctx context.Context, chargeID string, amount float64) error
}
