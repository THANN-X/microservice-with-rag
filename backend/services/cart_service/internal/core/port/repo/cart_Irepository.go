package repo

import (
	"cart_service/internal/core/domain"
	"context"
)

// CartCommandRepository — Write Side
// WHY FindOrCreateByUserID อยู่ใน Command Side?
//   - Lazy Creation เป็น Write Operation: อาจต้อง INSERT ถ้ายังไม่มีตะกร้า
//   - ทุก mutation (AddItem, RemoveItem, ...) ต้องการ cart_id ก่อน
type CartCommandRepository interface {
	// FindOrCreateByUserID — Token-Based Lazy Creation
	// ถ้ามีตะกร้าอยู่แล้วคืนกลับมา ถ้าไม่มีค่อยสร้างใหม่
	FindOrCreateByUserID(ctx context.Context, userID uint) (*domain.Cart, error)

	// UpsertItem — เพิ่ม item ถ้ายังไม่มี หรือบวกจำนวนถ้ามีอยู่แล้ว (ON CONFLICT DO UPDATE)
	// meta ใช้ denormalize product info ณ เวลาที่ add
	UpsertItem(ctx context.Context, cartID uint, variantID uint, quantity int, meta domain.CartItemMeta) error

	// SetItemQuantity — กำหนดจำนวน item ตรงๆ (ใช้ตอน PUT /cart/items/:variantId)
	SetItemQuantity(ctx context.Context, cartID uint, variantID uint, quantity int) error

	// RemoveItem — ลบ item ออกจากตะกร้า
	RemoveItem(ctx context.Context, cartID uint, variantID uint) error

	// ClearCart — ล้างของทั้งหมดในตะกร้า
	ClearCart(ctx context.Context, cartID uint) error
}

// CartQueryRepository — Read Side
type CartQueryRepository interface {
	// GetCartByUserID คืน ErrRecordNotFound ถ้าไม่พบตะกร้า (ไม่สร้างใหม่)
	GetCartByUserID(ctx context.Context, userID uint) (*domain.Cart, error)
}
