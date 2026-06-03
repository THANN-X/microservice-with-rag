package repository

import (
	"cart_service/internal/adapter/repository/postgres/entity"
	"cart_service/internal/core/domain"
	port "cart_service/internal/core/port/repo"
	"context"
	"database"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type cartRepository struct {
	*database.TxHelper
}

// NewCartRepository คืน Command และ Query repository จาก struct เดียว
// WHY struct เดียว implement ทั้งคู่?
//   - ขนาด service ยังเล็ก ไม่จำเป็นต้องแยก read/write DB
//   - Interface แยกไว้แล้วถ้าต้องการ scale ภายหลัง (เช่น read replica) แค่ implement ใหม่
func NewCartRepository(db *gorm.DB) (port.CartCommandRepository, port.CartQueryRepository) {
	r := &cartRepository{TxHelper: database.NewTxHelper(db)}
	return r, r
}

// ─── COMMAND SIDE ───────────────────────────────────────────────────────────

// FindOrCreateByUserID — Token-Based Lazy Creation
// WHY ใช้ First + Create แทน FirstOrCreate?
//   - FirstOrCreate ของ GORM ไม่ Preload associations
//   - เราต้องการ Items ด้วย → ต้องใช้ Preload("Items")
func (r *cartRepository) FindOrCreateByUserID(ctx context.Context, userID uint) (*domain.Cart, error) {
	var cartE entity.CartEntity

	err := r.GetDB(ctx).
		Preload("Items").
		Where("user_id = ?", userID).
		First(&cartE).Error

	if err == nil {
		return cartE.ToCartDomain(), nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// ไม่พบตะกร้า → สร้างใหม่ (Lazy Creation)
	newCart := entity.CartEntity{
		UserID: userID,
		Items:  []entity.CartItemEntity{},
	}
	if err := r.GetDB(ctx).Create(&newCart).Error; err != nil {
		return nil, err
	}

	return newCart.ToCartDomain(), nil
}

// UpsertItem — INSERT หรือบวกจำนวนถ้ามี (cart_id, variant_id) อยู่แล้ว
// WHY ใช้ clause.OnConflict?
//   - GORM รองรับ ON CONFLICT DO UPDATE ผ่าน clause.OnConflict
//   - gorm.Expr("cart_items.quantity + EXCLUDED.quantity") ทำ atomic accumulation
//   - metadata (product_name ฯลฯ) อัปเดตเมื่อ conflict เพื่อให้ข้อมูลล่าสุดเสมอ
func (r *cartRepository) UpsertItem(ctx context.Context, cartID uint, variantID uint, quantity int, meta domain.CartItemMeta) error {
	item := entity.CartItemEntity{
		CartID:      cartID,
		VariantID:   variantID,
		Quantity:    quantity,
		ProductName: meta.ProductName,
		VariantName: meta.VariantName,
		Price:       meta.Price,
		ImageURL:    meta.ImageURL,
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}
	return r.GetDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "cart_id"},
			{Name: "variant_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":     gorm.Expr("cart_items.quantity + EXCLUDED.quantity"),
			"product_name": gorm.Expr("EXCLUDED.product_name"),
			"variant_name": gorm.Expr("EXCLUDED.variant_name"),
			"price":        gorm.Expr("EXCLUDED.price"),
			"image_url":    gorm.Expr("EXCLUDED.image_url"),
			"updated_at":   gorm.Expr("NOW()"),
		}),
	}).Create(&item).Error
}

// SetItemQuantity — กำหนดจำนวน item ตรงๆ (ใช้ตอน PUT)
func (r *cartRepository) SetItemQuantity(ctx context.Context, cartID uint, variantID uint, quantity int) error {
	result := r.GetDB(ctx).
		Model(&entity.CartItemEntity{}).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		Updates(map[string]interface{}{
			"quantity":   quantity,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

// RemoveItem — ลบ item ด้วย GORM Delete + Where condition
func (r *cartRepository) RemoveItem(ctx context.Context, cartID uint, variantID uint) error {
	result := r.GetDB(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		Delete(&entity.CartItemEntity{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

// ClearCart — ลบ items ทั้งหมดของตะกร้าด้วย GORM Delete + Where
func (r *cartRepository) ClearCart(ctx context.Context, cartID uint) error {
	return r.GetDB(ctx).
		Where("cart_id = ?", cartID).
		Delete(&entity.CartItemEntity{}).Error
}

// ─── QUERY SIDE ─────────────────────────────────────────────────────────────

// GetCartByUserID คืน ErrRecordNotFound ถ้าไม่พบ (ไม่สร้างใหม่)
func (r *cartRepository) GetCartByUserID(ctx context.Context, userID uint) (*domain.Cart, error) {
	var cartE entity.CartEntity

	err := r.GetDB(ctx).Preload("Items").Where("user_id = ?", userID).First(&cartE).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRecordNotFound
		}
		return nil, err
	}

	return cartE.ToCartDomain(), nil
}
