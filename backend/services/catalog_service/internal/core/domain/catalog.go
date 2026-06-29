package domain

import "time"

// CatalogProduct คือ denormalized read model ที่สร้างจาก product_service events
// WHY: ลด latency ในการ query สำหรับหน้า customer-facing โดยเก็บข้อมูลที่ต้องใช้ไว้ใน document เดียว
// ข้อมูลอัปเดตผ่าน Kafka events อาจ lag เล็กน้อยอยู่บ้าง (eventual consistency)
type CatalogProduct struct {
	ID          string             // MongoDB _id (ObjectID hex string)
	ProductID   uint               // source product ID — unique lookup key
	Name        string
	Description string
	ImageURLs   []string
	Categories  []EmbeddedCategory
	Variants    []EmbeddedVariant
	IsActive    bool
	IsDeleted   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EmbeddedVariant struct {
	VariantID  uint
	Sku        string
	Name       string
	Price      float64
	Stock      int
	IsActive   bool
	ImageURLs  []string
	Attributes []VariantAttribute
}

type VariantAttribute struct {
	Key   string
	Value string
}

type EmbeddedCategory struct {
	CategoryID uint
	Name       string
	Slug       string
}

// ProductFilter กำหนด query params สำหรับ listing / search
type ProductFilter struct {
	Page        int
	Limit       int
	Search      string
	CategoryID  uint
	CategoryIDs []uint
	SortBy      string
	Order       string
}

// InboxMessage ใช้เก็บ messageID ที่ consume แล้วเพื่อ enforce idempotency
type InboxMessage struct {
	ID          string
	ConsumerID  string
	ProcessedAt time.Time
}
