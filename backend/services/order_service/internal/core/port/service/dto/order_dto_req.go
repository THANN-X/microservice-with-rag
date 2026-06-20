// WHAT: Request DTOs สำหรับ Order use cases
// WHY แยก DTO ออกจาก domain object?
//   - DTO คือ "input contract" ระหว่าง handler และ service
//   - Domain object อาจมี field ที่ client ไม่ควรส่งมา (e.g. ID, Status, CreatedAt)
//   - Validate tag บน DTO ป้องกัน bad input ก่อนถึง domain logic
package dto

// CreateOrderReq คือ request body สำหรับ POST /orders (customer places order)
type CreateOrderReq struct {
	Items           []CreateOrderItemReq `json:"items"            validate:"required,min=1,dive"`
	ShippingAddress ShippingAddressReq   `json:"shipping_address" validate:"required"`
	Note            string               `json:"note"`
}

// CreateOrderItemReq แทน 1 line item ใน order
// WHY ไม่มี UnitPrice แล้ว?
//   - ป้องกัน price tampering: แฮกเกอร์ส่ง unit_price=1 บาทแทนราคาจริง
//   - order_service ดึงราคาจาก catalog_service เอง (server-side price lookup)
//   - ผลพลอยได้: ได้ product_name + variant_name + image_url สำหรับ denormalization ด้วย
type CreateOrderItemReq struct {
	VariantID uint `json:"variant_id" validate:"required"`
	Quantity  int  `json:"quantity"   validate:"required,min=1"`
}

// ShippingAddressReq ที่อยู่จัดส่ง (ทุก field required)
type ShippingAddressReq struct {
	FullName    string `json:"full_name"    validate:"required"`
	Phone       string `json:"phone"        validate:"required"`
	AddressLine string `json:"address_line" validate:"required"`
	SubDistrict string `json:"sub_district" validate:"required"`
	District    string `json:"district"     validate:"required"`
	Province    string `json:"province"     validate:"required"`
	PostalCode  string `json:"postal_code"  validate:"required"`
}

// CancelOrderReq request body สำหรับ cancel order
type CancelOrderReq struct {
	Reason string `json:"reason" validate:"required"`
}

// HandleStockResultReq DTO สำหรับ HandleStockResult use case
// แปลงมาจาก events.StockReservedEvent ที่ message handler รับจาก Kafka
// WHY ใช้ DTO แทน events struct โดยตรง?
//   - Service layer ไม่ depend on events package (Hexagonal Architecture)
//   - Message handler รับผิดชอบ mapping (adapter role)
type HandleStockResultReq struct {
	OrderID   string
	MessageID string // Kafka message key (AggregateID) ใช้เป็น InboxEvent.ID
	Status    string // "SUCCESS" หรือ "FAILED"
}

// ProcessPaymentReq ลูกค้าส่ง payment token จาก frontend
type ProcessPaymentReq struct {
	// Token = PaymentMethod ID (pm_xxxx) จาก Stripe.js
	// WHY ไม่ required สำหรับ PromptPay? — PromptPay ไม่ต้องการ token จาก frontend
	//   frontend ส่ง payment_method=PROMPTPAY มา → backend สร้าง PaymentIntent เอง
	Token         string `json:"token"`
	PaymentMethod string `json:"payment_method" validate:"required"`
}

// PaymentWebhookReq ข้อมูลจาก payment gateway webhook
type PaymentWebhookReq struct {
	Signature string
	Payload   []byte
}
