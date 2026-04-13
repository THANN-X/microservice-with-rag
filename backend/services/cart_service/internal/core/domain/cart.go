package domain

import "time"

// Cart คือ Aggregate Root ของระบบตะกร้าสินค้า
// แต่ละ User จะมีตะกร้าได้ 1 ใบ (1-to-1) โดย UserID เป็น Unique Key
type Cart struct {
	ID        uint
	UserID    uint
	Items     []CartItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CartItem คือ item ภายในตะกร้า
// Composite PK = (CartID, VariantID) — แต่ละ variant มีได้แค่ 1 row ต่อตะกร้า
type CartItem struct {
	CartID    uint
	VariantID uint
	Quantity  int
	AddedAt   time.Time
	UpdatedAt time.Time
}

// NewCart สร้างตะกร้าใหม่สำหรับ user
func NewCart(userID uint) *Cart {
	return &Cart{
		UserID:    userID,
		Items:     []CartItem{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// AddItem เพิ่ม item หรือบวกจำนวนถ้า variant นั้นมีอยู่แล้ว (domain logic for in-memory)
func (c *Cart) AddItem(variantID uint, quantity int) {
	for i, item := range c.Items {
		if item.VariantID == variantID {
			c.Items[i].Quantity += quantity
			c.Items[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return
		}
	}
	c.Items = append(c.Items, CartItem{
		CartID:    c.ID,
		VariantID: variantID,
		Quantity:  quantity,
		AddedAt:   time.Now(),
		UpdatedAt: time.Now(),
	})
	c.UpdatedAt = time.Now()
}

// RemoveItem ลบ item ออกจากตะกร้าตาม variantID
func (c *Cart) RemoveItem(variantID uint) error {
	for i, item := range c.Items {
		if item.VariantID == variantID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrItemNotFound
}

// UpdateItemQuantity กำหนดจำนวน item ตรงๆ (ถ้า quantity <= 0 จะลบออก)
func (c *Cart) UpdateItemQuantity(variantID uint, quantity int) error {
	for i, item := range c.Items {
		if item.VariantID == variantID {
			if quantity <= 0 {
				c.Items = append(c.Items[:i], c.Items[i+1:]...)
				c.UpdatedAt = time.Now()
				return nil
			}
			c.Items[i].Quantity = quantity
			c.Items[i].UpdatedAt = time.Now()
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrItemNotFound
}

// Clear ล้างตะกร้าทั้งหมด
func (c *Cart) Clear() {
	c.Items = []CartItem{}
	c.UpdatedAt = time.Now()
}
