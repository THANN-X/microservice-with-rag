package repo

import (
	"catalog_service/internal/core/domain"
	"context"
	"time"
)

// CatalogWriteRepository — Write Side
// ข้อมูลถูก upsert/update จาก Kafka events เท่านั้น ไม่มี HTTP write endpoint
type CatalogWriteRepository interface {
	// Upsert สร้างหรืออัปเดต document ทั้งใบตาม product_id (idempotent)
	Upsert(ctx context.Context, product *domain.CatalogProduct) error

	// UpdateInfo อัปเดต name และ description
	UpdateInfo(ctx context.Context, productID uint, name, description string) error

	// UpdateVariantPrice อัปเดตราคาของ variant ที่ระบุ (ใช้ MongoDB arrayFilters)
	// occurredAt ใช้เป็น Timestamp Guard — อัปเดตเฉพาะเมื่อ event ใหม่กว่าค่าที่เก็บไว้
	UpdateVariantPrice(ctx context.Context, productID uint, variantID uint, newPrice float64, occurredAt time.Time) error

	// AddVariant เพิ่ม variant ใหม่เข้า document
	AddVariant(ctx context.Context, productID uint, variant domain.EmbeddedVariant) error

	// UpdateVariantStock อัปเดต stock ของ variant ที่ระบุ
	// occurredAt ใช้เป็น Timestamp Guard — อัปเดตเฉพาะเมื่อ event ใหม่กว่าค่าที่เก็บไว้
	UpdateVariantStock(ctx context.Context, productID uint, variantID uint, newStock int, occurredAt time.Time) error

	// UpdateProductImages แทนที่ image list ระดับ product
	UpdateProductImages(ctx context.Context, productID uint, imageURLs []string) error
	UpdateVariantImages(ctx context.Context, productID uint, variantID uint, imageURLs []string) error

	// UpdateCategories แทนที่ category list ทั้งหมดของ product (ใช้ filter สินค้าตามหมวดหมู่)
	UpdateCategories(ctx context.Context, productID uint, categories []domain.EmbeddedCategory) error

	// SetActive ตั้ง is_active ระดับ product (ซ่อน/แสดงสินค้าบนหน้าเว็บ)
	SetActive(ctx context.Context, productID uint, active bool) error

	// SetVariantActive ตั้ง is_active ระดับ variant ที่ระบุ
	// occurredAt ใช้เป็น Timestamp Guard — อัปเดตเฉพาะเมื่อ event ใหม่กว่าค่าที่เก็บไว้
	SetVariantActive(ctx context.Context, productID uint, variantID uint, active bool, occurredAt time.Time) error

	// MarkDeleted soft-delete — ตั้ง is_deleted=true, is_active=false
	MarkDeleted(ctx context.Context, productID uint) error
}

// CatalogReadRepository — Read Side
type CatalogReadRepository interface {
	// FindByProductID คืน product ที่ยังไม่ถูกลบตาม product source ID
	FindByProductID(ctx context.Context, productID uint) (*domain.CatalogProduct, error)

	// FindByVariantID คืน product และ variant ตาม variant ID
	FindByVariantID(ctx context.Context, variantID uint) (*domain.CatalogProduct, *domain.EmbeddedVariant, error)

	// FindAll คืนรายการ product แบบ paginated ตาม filter
	FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.CatalogProduct, int64, error)
}
