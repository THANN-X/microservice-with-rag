// WHAT: package events รวม Domain Event structs ทั้งหมดที่ใช้ส่งระหว่าง services ผ่าน Kafka
// WHY: ใช้ shared package แทนที่จะ define struct ใน service ตัวเอง เพื่อให้
//   - Producer และ Consumer ใช้ schema เดียวกันเสมอ (single source of truth)
//   - ลด risk ที่ field จะ mismatch กันระหว่าง services
//
// TODO: พิจารณาย้ายไปใช้ Protobuf หรือ Avro ร่วมกับ Schema Registry
//
//	เพื่อ enforce schema compatibility แบบ strict ในอนาคต
package events

import "time"

// WHAT: DomainEvent interface บังคับให้ทุก event struct มี EventName()
// WHY: ใช้ type-safe dispatch และ logging โดยไม่ต้อง hardcode string
type DomainEvent interface {
	EventName() string
}

// WHAT: EventTypeHeader ใช้สำหรับ peek event type จาก JSON body
//        โดยไม่ต้อง unmarshal payload ทั้งก้อน
// WHY: Consumer ต้องรู้ event type ก่อนเพื่อเลือก struct ที่จะ unmarshal
//      Primary path คืออ่านจาก Kafka Header "EventType"
//      Fallback path คืออ่านจาก field "event_type" ใน body นี้
// TODO: ถ้าตกลงกันได้ว่า producer ทุกตัวใส่ Header เสมอ ให้ลบ fallback นี้ออกได้
type EventTypeHeader struct {
	EventType string `json:"event_type"`
}

type ProductPriceChangedEvent struct {
	ProductID  uint
	VariantID  uint
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

// WHAT: StockReservedEvent ส่งออกหลังจาก product_service reserve stock สำเร็จ
// WHY: แจ้ง order_service ว่า stock ถูกจับจองแล้ว ให้ดำเนินการขั้นตอนต่อไปของ Saga ได้
//      MessageID ใช้ทำ Idempotency ฝั่ง consumer
type StockReservedEvent struct {
	OrderID    string         `json:"order_id"`
	MessageID  string         `json:"message_id"`
	Status     string         `json:"status"`
	Items      []ReservedItem `json:"item_reserved"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// WHAT: StockReleasedEvent ส่งออกหลังจาก product_service release stock สำเร็จ (Saga rollback)
// WHY: แจ้ง order_service ว่า stock ถูกคืนแล้ว ใช้ใน Compensating Transaction
//      MessageID ใช้ทำ Idempotency ฝั่ง consumer
type StockReleasedEvent struct {
	OrderID    string         `json:"order_id"`
	MessageID  string         `json:"message_id"`
	Status     string         `json:"status"`
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

// Event: Product-level images replaced
type ProductImagesUpdatedEvent struct {
	ProductID  uint      `json:"product_id"`
	ImageURLs  []string  `json:"image_urls"`
	UpdatedBy  uint      `json:"updated_by"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Event: Variant-level images replaced (e.g. colour-specific photos)
type ProductVariantImagesUpdatedEvent struct {
	ProductID  uint      `json:"product_id"`
	VariantID  uint      `json:"variant_id"`
	ImageURLs  []string  `json:"image_urls"`
	UpdatedBy  uint      `json:"updated_by"`
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

// CategoryKV เป็น category snapshot (id + name + slug) ที่ฝังไปกับ event
// เพื่อให้ catalog_service embed ลง document ได้โดยไม่ต้อง query product_service กลับ
type CategoryKV struct {
	CategoryID uint   `json:"category_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
}

// Event: Product categories set/replaced (ตอนสร้าง product หรือแก้ไข general info)
// WHY ต้องมี? — catalog_service ใช้ categories ในการ filter สินค้า (category_id)
//
//	ถ้าไม่ sync → สินค้าจะ filter ตาม category ใน catalog ไม่เจอ
type ProductCategoriesUpdatedEvent struct {
	ProductID  uint         `json:"product_id"`
	Categories []CategoryKV `json:"categories"`
	OccurredAt time.Time    `json:"occurred_at"`
}

// Event: Product visibility toggled (active/inactive)
// WHY ต้องมี? — ถ้า admin ซ่อนสินค้า (is_active=false) ใน product_service แต่ไม่ส่ง event
//
//	catalog (read model) จะยังโชว์สินค้านั้นบนหน้าเว็บต่อไป → ข้อมูลไม่ตรงกัน
type ProductActiveChangedEvent struct {
	ProductID  uint      `json:"product_id"`
	IsActive   bool      `json:"is_active"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Event: Variant visibility toggled (active/inactive)
type ProductVariantActiveChangedEvent struct {
	ProductID  uint      `json:"product_id"`
	VariantID  uint      `json:"variant_id"`
	IsActive   bool      `json:"is_active"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Event: Stock changed to an absolute value (reserve/release/adjust รวมเป็น event เดียว)
// WHY absolute แทน delta? — idempotent + ทน message ซ้ำ/หาย/มาผิดลำดับ (เซฟค่าจริงทับทื่อๆ)
//
//	catalog ฟัง event เดียวนี้เพื่อ sync stock ทุกกรณี (ซื้อ/คืน/ปรับ)
type StockUpdatedEvent struct {
	ProductID  uint      `json:"product_id"`
	VariantID  uint      `json:"variant_id"`
	NewStock   int       `json:"new_stock"`
	OccurredAt time.Time `json:"occurred_at"`
}

// WHAT: OrderCreatedEvent คือ event ที่ order_service raise เมื่อ Order ถูกสร้างสำเร็จ
// WHY: product_service ใช้ Items (VariantID+Qty) สำหรับ reserve stock (Saga step 1)
//      order_history_service ใช้ข้อมูลทั้งหมดสร้าง denormalized read model
type OrderCreatedEvent struct {
	OrderID         string               `json:"order_id"`
	CustomerID      uint                 `json:"customer_id"`
	Items           []OrderItem          `json:"items"`
	TotalAmount     float64              `json:"total_amount"`
	ShippingAddress EventShippingAddress `json:"shipping_address"`
	Note            string               `json:"note"`
	OccurredAt      time.Time            `json:"occurred_at"`
}

// WHAT: OrderCancelledEvent คือ inbound event ที่ product_service รับจาก order_service
// WHY: เมื่อ order ถูก cancel product_service ต้อง release stock กลับ (Compensating Transaction)
//      Reason field เก็บไว้เพื่อ audit log ว่า cancel เพราะอะไร (timeout, user cancel, payment failed)
// TODO: พิจารณาแยก PaymentFailedEvent ออกมาต่างหาก
//       ถ้า business logic การ release stock ต่างกันระหว่าง user cancel กับ payment failed
type OrderCancelledEvent struct {
	OrderID    string      `json:"order_id"`
	Items      []OrderItem `json:"items"`
	Reason     string      `json:"reason"`
	OccurredAt time.Time   `json:"occurred_at"`
}

// WHAT: OrderItem ระบุ variant และ quantity ที่ต้องการ reserve/release
// WHY: ใช้ VariantID แทน ProductID เพราะ stock จัดการระดับ variant (size/color)
//      1 product มีได้หลาย variant แต่ละตัวมี stock แยกกัน
//      UnitPrice ใช้สำหรับ order_history read model (product_service ไม่จำเป็นต้องใช้)
type OrderItem struct {
	VariantID uint    `json:"variant_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price,omitempty"`
}

// EventShippingAddress ใช้ส่งข้อมูลที่อยู่จัดส่งผ่าน event
type EventShippingAddress struct {
	FullName    string `json:"full_name"`
	Phone       string `json:"phone"`
	AddressLine string `json:"address_line"`
	SubDistrict string `json:"sub_district"`
	District    string `json:"district"`
	Province    string `json:"province"`
	PostalCode  string `json:"postal_code"`
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

func (e ProductImagesUpdatedEvent) EventName() string {
	return "PRODUCT_IMAGES_UPDATED"
}

func (e ProductVariantImagesUpdatedEvent) EventName() string {
	return "PRODUCT_VARIANT_IMAGES_UPDATED"
}

func (e StockAdjustedEvent) EventName() string {
	return "STOCK_ADJUSTED"
}

func (e ProductCategoriesUpdatedEvent) EventName() string {
	return "PRODUCT_CATEGORIES_UPDATED"
}

func (e ProductActiveChangedEvent) EventName() string {
	return "PRODUCT_ACTIVE_CHANGED"
}

func (e ProductVariantActiveChangedEvent) EventName() string {
	return "PRODUCT_VARIANT_ACTIVE_CHANGED"
}

func (e StockUpdatedEvent) EventName() string {
	return "STOCK_UPDATED"
}

// ─── Order Events ─────────────────────────────────────────────────────────────
// WHAT: Order Domain Events ใช้สำหรับ Choreography-based Saga
//       order_service เป็น producer, product_service เป็น consumer (และในอนาคต notification, payment)
// WHY แยก OrderItem (events package) จาก domain.OrderItem (order_service)?
//   - events package เป็น shared contract → ต้องไม่ depend on any service-specific domain model
//   - OrderItem ที่นี่มีแค่ field ที่ downstream services (product_service) ต้องการ

// OrderCreatedEvent raised by order_service เมื่อ Order ถูกสร้างสำเร็จ
// WHY: triggers product_service ให้ reserve stock (Saga step 1)
//      MessageID ใช้เป็น InboxEvent.ID ที่ product_service เพื่อทำ exactly-once semantics
func (e OrderCreatedEvent) EventName() string { return "ORDER_CREATED" }

// OrderCancelledEvent raised by order_service เมื่อ Order (ที่ stock เคยถูก reserved) ถูก cancel
// WHY: triggers product_service ให้ release stock (Compensating Transaction)
//      MessageID ต้องเป็น UUID ใหม่ทุกครั้งที่ raise ไม่ซ้ำกับ MessageID ของ reserve event
//      เพื่อให้ product_service ประมวลผล release ได้อย่าง idempotent แยกจาก reserve
func (e OrderCancelledEvent) EventName() string { return "ORDER_CANCELLED" }

// OrderConfirmedEvent raised by order_service เมื่อ stock reservation สำเร็จ
// WHY: downstream services (notification, payment) สามารถ consume event นี้เพื่อดำเนินการต่อ
//      เช่น ส่ง email confirm หรือ trigger payment flow
type OrderConfirmedEvent struct {
	OrderID     string    `json:"order_id"`
	CustomerID  uint      `json:"customer_id"`
	TotalAmount float64   `json:"total_amount"`
	OccurredAt  time.Time `json:"occurred_at"`
}

func (e OrderConfirmedEvent) EventName() string { return "ORDER_CONFIRMED" }

type OrderPaidEvent struct {
	OrderID     string    `json:"order_id"`
	CustomerID  uint      `json:"customer_id"`
	TotalAmount float64   `json:"total_amount"`
	OccurredAt  time.Time `json:"occurred_at"`
}

func (e OrderPaidEvent) EventName() string { return "ORDER_PAID" }
