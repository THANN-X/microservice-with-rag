package port

import (
	"context"
	"product_service/internal/core/domain"
)

// COMMAND REPOSITORY (Connect Master DB / Postgres)
type ProductCommandRepository interface {
	// Transaction Helpers
	TransactionManager

	// Product Management
	CreateProduct(ctx context.Context, product *domain.Product) error
	GetProductByID(ctx context.Context, id uint) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) error
	// DeleteProduct รับ *domain.Product (ไม่ใช่ id) เพื่อให้ repo auto-save domain events ได้
	// caller ต้องเรียก product.MarkAsDeleted() ก่อนส่งเข้ามา
	DeleteProduct(ctx context.Context, product *domain.Product) error

	AddVariant(ctx context.Context, variant *domain.ProductVariant) error
	UpdateStock(ctx context.Context, variantID uint, newStock int) error

	// Active / Inactive Management
	// ใช้ targeted UPDATE แทน Save ทั้งหลัง เพื่อหลีกเลี่ยง race condition และ unnecessary field updates
	SetProductActive(ctx context.Context, productID uint, active bool) error
	SetVariantActive(ctx context.Context, variantID uint, active bool) error

	// Stock Management (Atomic Update)
	// SQL: UPDATE variants SET stock = stock - ? WHERE id = ? AND stock >= ?
	DecreaseStock(ctx context.Context, variantID uint, qty int) error
	IncreaseStock(ctx context.Context, variantID uint, qty int) error

	SaveDomainEvents(ctx context.Context, product *domain.Product) error
}

// QUERY REPOSITORY (Connect Read Replica, Redis, Or Postgres nor)
type ProductQueryRepository interface {
	FindByID(ctx context.Context, id uint) (*domain.Product, error)

	FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error)
}
