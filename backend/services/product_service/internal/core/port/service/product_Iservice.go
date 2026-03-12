package port

import (
	"context"
	dto "product_service/internal/core/port/service/dto"
)

type ProductCommandService interface {
	// Product Lifecycle Management
	CreateProduct(ctx context.Context, UserID uint, req *dto.CreateProductReq) error
	UpdateProductGeneralInfo(ctx context.Context, userID uint, req *dto.UpdateProductGeneralInfoReq) error
	DeleteProduct(ctx context.Context, userID uint, productID uint) error

	// Active / Inactive Management
	// ลดความซับซ้อนโดยใช้ bool แทนการสร้างสอง method (Activate/Deactivate)
	SetProductActive(ctx context.Context, userID uint, productID uint, active bool) error
	SetVariantActive(ctx context.Context, userID uint, productID uint, variantID uint, active bool) error

	// Variant Management (Admin)
	AddVariant(ctx context.Context, userID uint, req *dto.AddVariantReq) error
	UpdateVariantPrice(ctx context.Context, userID uint, req *dto.UpdateVariantPriceReq) error

	// Stock Management (Admin Manual Adjust)
	AdjustStock(ctx context.Context, userID uint, req *dto.AdjustStockReq) error

	// Stock Management (Consumer / System)
	ReserveStock(ctx context.Context, req *dto.ReserveStockReq) error
	ReleaseStock(ctx context.Context, req *dto.ReserveStockReq) error
}

// QUERY SERVICE (Read Side - For Admin Backoffice)
type ProductQueryService interface {
	GetProductByID(ctx context.Context, id uint) (*dto.ProductRes, error)

	ListProducts(ctx context.Context, req *dto.ListProductReq) (*dto.ProductListRes, error)
}
