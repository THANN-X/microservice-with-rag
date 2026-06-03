package entity

import (
	"cart_service/internal/core/domain"
	"time"
)

// CartEntity maps to "carts" table
// WHY ไม่ใช้ gorm.Model?
//   - ไม่ต้องการ soft delete สำหรับตะกร้า (hard delete เมื่อ user ลบ)
//   - ลด overhead ของ deleted_at column
//   - GORM ยังคง auto-manage CreatedAt/UpdatedAt อยู่
type CartEntity struct {
	ID        uint             `gorm:"primaryKey;autoIncrement"`
	UserID    uint             `gorm:"uniqueIndex;not null"`
	Items     []CartItemEntity `gorm:"foreignKey:CartID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CartEntity) TableName() string {
	return "carts"
}

func (c *CartEntity) ToCartDomain() *domain.Cart {
	items := make([]domain.CartItem, len(c.Items))
	for i, item := range c.Items {
		items[i] = domain.CartItem{
			CartID:      item.CartID,
			VariantID:   item.VariantID,
			Quantity:    item.Quantity,
			ProductName: item.ProductName,
			VariantName: item.VariantName,
			Price:       item.Price,
			ImageURL:    item.ImageURL,
			AddedAt:     item.AddedAt,
			UpdatedAt:   item.UpdatedAt,
		}
	}
	return &domain.Cart{
		ID:        c.ID,
		UserID:    c.UserID,
		Items:     items,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// CartItemEntity maps to "cart_items" table
// Composite PK = (cart_id, variant_id) — 1 variant ต่อ 1 ตะกร้าเท่านั้น
// WHY ไม่มี gorm.Model?
//   - Composite PK ขัดแย้งกับ auto-increment ID ของ gorm.Model
//   - cart_items ถูก delete จริงๆ ไม่ต้อง soft-delete

// ProductName/VariantName/Price/ImageURL เป็น denormalized snapshot ณ เวลาที่ add
type CartItemEntity struct {
	CartID      uint      `gorm:"primaryKey"`
	VariantID   uint      `gorm:"primaryKey"`
	Quantity    int       `gorm:"not null"`
	ProductName string    `gorm:"type:text;default:''"`
	VariantName string    `gorm:"type:text;default:''"`
	Price       float64   `gorm:"type:numeric(12,2);default:0"`
	ImageURL    string    `gorm:"type:text;default:''"`
	AddedAt     time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (CartItemEntity) TableName() string {
	return "cart_items"
}
