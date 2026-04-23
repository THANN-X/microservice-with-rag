package repo

import (
	"catalog_service/internal/core/domain"
	"context"
)

// CatalogWriteRepository — Write Side
// ข้อมูลถูก upsert/update จาก Kafka events เท่านั้น ไม่มี HTTP write endpoint
type CatalogWriteRepository interface {
	// Upsert สร้างหรืออัปเดต document ทั้งใบตาม product_id (idempotent)
	Upsert(ctx context.Context, product *domain.CatalogProduct) error

	// UpdateInfo อัปเดต name และ description
	UpdateInfo(ctx context.Context, productID uint, name, description string) error

	// UpdateVariantPrice อัปเดตราคาของ variant ที่ระบุ (ใช้ MongoDB arrayFilters)
	UpdateVariantPrice(ctx context.Context, productID uint, variantID uint, newPrice float64) error

	// AddVariant เพิ่ม variant ใหม่เข้า document
	AddVariant(ctx context.Context, productID uint, variant domain.EmbeddedVariant) error

	// UpdateVariantStock อัปเดต stock ของ variant ที่ระบุ
	UpdateVariantStock(ctx context.Context, productID uint, variantID uint, newStock int) error

	// UpdateProductImages แทนที่ image list ระดับ product
	UpdateProductImages(ctx context.Context, productID uint, imageURLs []string) error

	// UpdateVariantImages แทนที่ image list ของ variant ที่ระบุ
	UpdateVariantImages(ctx context.Context, productID uint, variantID uint, imageURLs []string) error

	// MarkDeleted soft-delete — ตั้ง is_deleted=true, is_active=false
	MarkDeleted(ctx context.Context, productID uint) error
}

// CatalogReadRepository — Read Side
type CatalogReadRepository interface {
	// FindByProductID คืน product ที่ยังไม่ถูกลบตาม product source ID
	FindByProductID(ctx context.Context, productID uint) (*domain.CatalogProduct, error)

	// FindAll คืนรายการ product แบบ paginated ตาม filter
	FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.CatalogProduct, int64, error)
}
