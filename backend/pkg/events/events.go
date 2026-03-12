package events

import "time"

type DomainEvent interface {
	EventName() string
}

// สร้าง Struct นี้ไว้เพื่อ "แอบดู" Type โดยเฉพาะ
// (ตั้งชื่อ field ให้ตรงกับ JSON ที่ส่งมานะครับ เช่น "event_type" หรือ "type")
type EventTypeHeader struct {
	EventType string `json:"event_type"`
}

type ProductPriceChangedEvent struct {
	ProductID  uint
	OldPrice   float64
	NewPrice   float64
	OccurredAt time.Time
}

type ProductInfoUpdatedEvent struct {
	ProductID   uint
	Name        string
	Description string
}

type ProductCreatedEvent struct {
	ProductID uint
	Name      string
	// ส่งข้อมูลเท่าที่จำเป็นสำหรับ Consumer (เช่น Search Service)
	Description string
	CreatedBy   uint
	OccurredAt  time.Time
}

type StockReservedEvent struct {
	OrderID   string `json:"order_id"`
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	// Items       interface{} `json:"items"` // หรือระบุ Type ชัดเจนถ้าทำได้
	Items      []ReservedItem `json:"item_reserved"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type StockReleasedEvent struct {
	OrderID   string `json:"order_id"`
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	// Items      interface{} `json:"items"`
	Items      []ReservedItem `json:"item_released"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type ReservedItem struct {
	VariantID uint `json:"variant_id"`
	Qty       int  `json:"qty"`
}

// Event: Add New Variant
type ProductVariantAddedEvent struct {
	ProductID  uint          `json:"product_id"`
	VariantID  uint          `json:"variant_id"`
	Sku        string        `json:"sku"`
	Name       string        `json:"name"`
	Price      float64       `json:"price"`
	Stock      int           `json:"stock"`
	Attributes []AttributeKV `json:"attributes"` // Key-Value pair
	OccurredAt time.Time     `json:"occurred_at"`
}

// Helper struct for Attribute Key-Value pair in events
type AttributeKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Event: Soft Delete
type ProductDeletedEvent struct {
	ProductID  uint      `json:"product_id"`
	DeletedBy  uint      `json:"deleted_by"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Event: Manual Adjustment
type StockAdjustedEvent struct {
	ProductID  uint      `json:"product_id"`
	VariantID  uint      `json:"variant_id"`
	OldStock   int       `json:"old_stock"`
	NewStock   int       `json:"new_stock"`
	Reason     string    `json:"reason"`
	AdjustedBy uint      `json:"adjusted_by"`
	OccurredAt time.Time `json:"occurred_at"`
}

// สมมติว่า Order Service ส่ง Event หน้าตาแบบนี้มา
type OrderCreatedEvent struct {
	OrderID string      `json:"order_id"`
	Items   []OrderItem `json:"items"`
	// ... fields อื่นๆ
}

type OrderItem struct {
	ProductID uint `json:"product_id"` // หรือ VariantID แล้วแต่ตกลง
	Quantity  int  `json:"quantity"`
}

// Implement Interface
func (e ProductPriceChangedEvent) EventName() string {
	return "PRODUCT_PRICE_CHANGED"
}

func (e ProductInfoUpdatedEvent) EventName() string {
	return "PRODUCT_INFO_UPDATED"
}

func (e ProductCreatedEvent) EventName() string {
	return "PRODUCT_CREATED"
}

func (e StockReservedEvent) EventName() string {
	return "STOCK_RESERVED"
}

func (e StockReleasedEvent) EventName() string {
	return "STOCK_RELEASED"
}

func (e ProductVariantAddedEvent) EventName() string {
	return "PRODUCT_VARIANT_ADDED"
}

func (e ProductDeletedEvent) EventName() string {
	return "PRODUCT_DELETED"
}

func (e StockAdjustedEvent) EventName() string {
	return "STOCK_ADJUSTED"
}
