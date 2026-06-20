// WHAT: Response DTOs สำหรับ Order use cases
// WHY แยก Response DTO จาก domain object?
//   - domain.Order อาจมี field ที่ไม่ควรส่งหา client (เช่น domainEvents)
//   - Response DTO คือ "output contract" → เปลี่ยน API response โดยไม่กระทบ domain
//   - Subtotal บน item เป็น derived field (computed สำหรับ client convenience)
package dto

import "time"

// OrderRes คือ Order representation สำหรับ API response
type OrderRes struct {
	ID              string          `json:"id"`
	CustomerID      uint            `json:"customer_id"`
	Status          string          `json:"status"`
	TotalAmount     float64         `json:"total_amount"`
	Items           []OrderItemRes  `json:"items"`
	ShippingAddress ShippingAddressRes `json:"shipping_address"`
	Note            string          `json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// OrderItemRes แทน 1 line item ใน order response
type OrderItemRes struct {
	ID          string  `json:"id"`
	VariantID   uint    `json:"variant_id"`
	ProductName string  `json:"product_name"`  // Snapshot ณ เวลาสั่ง
	VariantName string  `json:"variant_name"`  // Snapshot ณ เวลาสั่ง
	ImageURL    string  `json:"image_url"`     // Snapshot ณ เวลาสั่ง
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"` // computed: UnitPrice * Quantity
}

// ShippingAddressRes ที่อยู่จัดส่ง (ส่งกลับให้ client ครบ)
type ShippingAddressRes struct {
	FullName    string `json:"full_name"`
	Phone       string `json:"phone"`
	AddressLine string `json:"address_line"`
	SubDistrict string `json:"sub_district"`
	District    string `json:"district"`
	Province    string `json:"province"`
	PostalCode  string `json:"postal_code"`
}

type PaymentRes struct {
	ID              string     `json:"id"`
	OrderID         string     `json:"order_id"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	Gateway         string     `json:"gateway"`
	GatewayChargeID string     `json:"gateway_charge_id,omitempty"`
	PaymentMethod   string     `json:"payment_method"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	// ClientSecret ใช้บน frontend สำหรับ 3DS (confirm modal)
	// ไม่เก็บใน DB — ส่งจาก gateway response ตรงๆ
	ClientSecret string `json:"client_secret,omitempty"`
	// QRImageURL คือ URL รูป QR PromptPay — ส่งตรงจาก Stripe next_action
	QRImageURL   string `json:"qr_image_url,omitempty"`
}
