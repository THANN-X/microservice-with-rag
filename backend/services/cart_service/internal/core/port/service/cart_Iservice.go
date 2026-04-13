package port

import (
	"context"
	dto "cart_service/internal/core/port/service/dto"
)

// CartCommandService — Write Side (ทุก operation ใช้ userID จาก JWT Token)
type CartCommandService interface {
	// AddItem เพิ่ม item เข้าตะกร้า ถ้าตะกร้ายังไม่มีจะสร้างให้อัตโนมัติ (Lazy Creation)
	AddItem(ctx context.Context, userID uint, req *dto.AddCartItemReq) (*dto.CartRes, error)

	// RemoveItem ลบ item ออกจากตะกร้า
	RemoveItem(ctx context.Context, userID uint, variantID uint) (*dto.CartRes, error)

	// UpdateItemQuantity อัพเดตจำนวน item (quantity=0 จะลบ item ออก)
	UpdateItemQuantity(ctx context.Context, userID uint, req *dto.UpdateCartItemReq) (*dto.CartRes, error)

	// ClearCart ล้างตะกร้าทั้งหมด
	ClearCart(ctx context.Context, userID uint) error
}

// CartQueryService — Read Side
type CartQueryService interface {
	// GetCart คืนตะกร้าของ user (ถ้ายังไม่มีตะกร้าจะคืน empty cart โดยไม่สร้าง)
	GetCart(ctx context.Context, userID uint) (*dto.CartRes, error)
}
