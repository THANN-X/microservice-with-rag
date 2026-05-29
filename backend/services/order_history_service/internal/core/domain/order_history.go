package domain

import "time"

// OrderHistory คือ denormalized read model สร้างจาก order_service events
// ข้อมูลอัปเดตผ่าน Kafka events (eventual consistency)
type OrderHistory struct {
	ID              string
	OrderID         string
	CustomerID      uint
	Status          string
	TotalAmount     float64
	Items           []OrderHistoryItem
	ShippingAddress ShippingAddress
	Note            string
	CancelReason    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderHistoryItem struct {
	VariantID uint
	Quantity  int
	UnitPrice float64
}

type ShippingAddress struct {
	FullName    string
	Phone       string
	AddressLine string
	SubDistrict string
	District    string
	Province    string
	PostalCode  string
}

type OrderHistoryFilter struct {
	Page       int
	Limit      int
	CustomerID uint
	Status     string
}

// OrderHistoryAdminFilter ใช้สำหรับ admin query ทุก order (ไม่ filter ตาม customerID)
type OrderHistoryAdminFilter struct {
	Page   int
	Limit  int
	Status string
}

// InboxMessage ใช้เก็บ messageID ที่ consume แล้วเพื่อ enforce idempotency
type InboxMessage struct {
	ID          string
	ConsumerID  string
	ProcessedAt time.Time
}
