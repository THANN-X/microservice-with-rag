package port

import (
	"catalog_service/internal/core/port/service/dto"
	"context"
	"events"
)

// CatalogCommandService — รับ domain events จาก Kafka แล้วอัปเดต read model ใน MongoDB
// ทุก handler ต้องตรวจ idempotency ก่อนเสมอ
type CatalogCommandService interface {
	HandleProductCreated(ctx context.Context, messageID string, evt *events.ProductCreatedEvent) error
	HandleProductInfoUpdated(ctx context.Context, messageID string, evt *events.ProductInfoUpdatedEvent) error
	HandleProductPriceChanged(ctx context.Context, messageID string, evt *events.ProductPriceChangedEvent) error
	HandleProductVariantAdded(ctx context.Context, messageID string, evt *events.ProductVariantAddedEvent) error
	HandleProductDeleted(ctx context.Context, messageID string, evt *events.ProductDeletedEvent) error
	HandleStockAdjusted(ctx context.Context, messageID string, evt *events.StockAdjustedEvent) error
}

// CatalogQueryService — Read Side ให้ BFF ใช้ query สินค้าสำหรับลูกค้า
type CatalogQueryService interface {
	// SearchProducts ค้นหาและ list สินค้าพร้อม pagination
	SearchProducts(ctx context.Context, req *dto.SearchProductsReq) (*dto.ProductListRes, error)

	// GetProductByID ดึงข้อมูลสินค้าเดียวตาม source product ID
	GetProductByID(ctx context.Context, productID uint) (*dto.CatalogProductRes, error)
}
