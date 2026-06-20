// WHAT: GORM entity ที่แมปกับตาราง orders และ order_items ใน DB
// WHY แยก Entity ออกจาก Domain Object?
//   - Domain Object ไม่ควรรู้จัก GORM tags หรือโครงสร้าง DB (Clean Architecture)
//   - ถ้ารวมกัน GORM จะรั่ว domain model ทำให้ Unit Test ยาก (ต้อง mock DB)
//   - Entity คือ "DB representation", Domain คือ "business representation"
//
// WHY ไม่ใช้ gorm.Model (uint auto-increment)?
//   - Order ใช้ UUID เป็น PK → ต้องกำหนด type:uuid เอง
//   - gorm.Model มี DeletedAt ที่เราไม่ต้องการ (ใช้ Status field สำหรับ lifecycle แทน)
//
// WHY embed ShippingAddress fields โดยตรงใน OrderEntity แทน separate table?
//   - Address เป็น Value Object → ไม่มี identity ของตัวเอง → ไม่ต้องการ separate table
//   - Denormalize ลง orders table → ลด JOIN ใน query
//   - ถ้ามี Address ที่ใช้ร่วมกันหลาย Orders → พิจารณา normalize ใหม่ (TODO)
package entity

import (
	"order_service/internal/core/domain"
	"time"
)

// OrderEntity maps to "orders" table
type OrderEntity struct {
	ID          string           `gorm:"primaryKey;type:varchar(36)"` // UUID
	CustomerID  uint             `gorm:"not null;index"`
	Status      string           `gorm:"type:varchar(50);not null;default:'PENDING';index"` // index เพื่อ filter by status เร็ว
	TotalAmount float64          `gorm:"type:decimal(12,2);not null;default:0"`
	Items       []OrderItemEntity `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE;"` // CASCADE: ลบ order → ลบ items
	// ShippingAddress embedded fields (denormalized)
	ShipFullName    string `gorm:"type:varchar(255);not null"`
	ShipPhone       string `gorm:"type:varchar(30);not null"`
	ShipAddressLine string `gorm:"type:text;not null"`
	ShipSubDistrict string `gorm:"type:varchar(100);not null"`
	ShipDistrict    string `gorm:"type:varchar(100);not null"`
	ShipProvince    string `gorm:"type:varchar(100);not null"`
	ShipPostalCode  string `gorm:"type:varchar(20);not null"`
	Note            string `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (OrderEntity) TableName() string { return "orders" }

// OrderItemEntity maps to "order_items" table
// WHY ไม่มี gorm.Model (no soft-delete, no uint PK)?
//   - Item ใช้ UUID เหมือน Order
//   - Items ไม่ถูก soft-delete แยก: ลบพร้อม Order (CASCADE)
type OrderItemEntity struct {
	ID          string  `gorm:"primaryKey;type:varchar(36)"`
	OrderID     string  `gorm:"type:varchar(36);not null;index"` // FK → orders.id
	VariantID   uint    `gorm:"not null;index"`                 // Reference to product_service variant
	Quantity    int     `gorm:"not null"`
	UnitPrice   float64 `gorm:"type:decimal(10,2);not null"`    // Price snapshot (server-side จาก catalog)
	ProductName string  `gorm:"type:varchar(255);not null;default:''"` // Denormalized: ชื่อสินค้า ณ เวลาสั่ง
	VariantName string  `gorm:"type:varchar(255);not null;default:''"` // Denormalized: ชื่อ variant ณ เวลาสั่ง
	ImageURL    string  `gorm:"type:text;not null;default:''"` // Denormalized: รูปสินค้า ณ เวลาสั่ง
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (OrderItemEntity) TableName() string { return "order_items" }

// ─── Bidirectional Mappers ─────────────────────────────────────────────────────

// ToOrderDomain แปลง GORM entity → domain.Order (Anti-Corruption Layer)
func (e *OrderEntity) ToOrderDomain() *domain.Order {
	if e == nil {
		return nil
	}

	items := make([]domain.OrderItem, len(e.Items))
	for i, item := range e.Items {
		items[i] = domain.OrderItem{
			ID:          item.ID,
			OrderID:     item.OrderID,
			VariantID:   item.VariantID,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			ProductName: item.ProductName,
			VariantName: item.VariantName,
			ImageURL:    item.ImageURL,
		}
	}

	return &domain.Order{
		ID:          e.ID,
		CustomerID:  e.CustomerID,
		Status:      domain.OrderStatus(e.Status),
		TotalAmount: e.TotalAmount,
		Items:       items,
		ShippingAddress: domain.ShippingAddress{
			FullName:    e.ShipFullName,
			Phone:       e.ShipPhone,
			AddressLine: e.ShipAddressLine,
			SubDistrict: e.ShipSubDistrict,
			District:    e.ShipDistrict,
			Province:    e.ShipProvince,
			PostalCode:  e.ShipPostalCode,
		},
		Note:      e.Note,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// ToOrderEntity แปลง domain.Order → GORM entity ก่อน INSERT หรือ UPDATE
// WHY ต้อง set ID ด้วย?
//   - ถ้า ID ว่าง GORM ไม่รู้จะ INSERT หรือ UPDATE → กำหนด ID จาก domain ก่อน
func ToOrderEntity(o *domain.Order) *OrderEntity {
	if o == nil {
		return nil
	}

	items := make([]OrderItemEntity, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemEntity{
			ID:          item.ID,
			OrderID:     item.OrderID,
			VariantID:   item.VariantID,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			ProductName: item.ProductName,
			VariantName: item.VariantName,
			ImageURL:    item.ImageURL,
		}
	}

	return &OrderEntity{
		ID:              o.ID,
		CustomerID:      o.CustomerID,
		Status:          string(o.Status),
		TotalAmount:     o.TotalAmount,
		Items:           items,
		ShipFullName:    o.ShippingAddress.FullName,
		ShipPhone:       o.ShippingAddress.Phone,
		ShipAddressLine: o.ShippingAddress.AddressLine,
		ShipSubDistrict: o.ShippingAddress.SubDistrict,
		ShipDistrict:    o.ShippingAddress.District,
		ShipProvince:    o.ShippingAddress.Province,
		ShipPostalCode:  o.ShippingAddress.PostalCode,
		Note:            o.Note,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}
