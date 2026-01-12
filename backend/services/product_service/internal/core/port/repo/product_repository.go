package port

import (
	"context"
	"product_service/internal/core/domain"
)

// =================================================================
// COMMAND REPOSITORY (ต่อกับ Master DB / Postgres)
// =================================================================
type ProductCommandRepository interface {
	// Transaction Helpers
	// เรามักต้องใช้ Transaction เวลาตัดสต็อกหรือสร้าง Product ก้อนใหญ่
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error

	// Product Management
	CreateProduct(ctx context.Context, product *domain.Product) error
	UpdateProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, id uint) error

	// Stock Management (Atomic Update)
	// SQL: UPDATE variants SET stock = stock - ? WHERE id = ? AND stock >= ?
	DecreaseStock(ctx context.Context, variantID uint, qty int) error
	IncreaseStock(ctx context.Context, variantID uint, qty int) error

	// Idempotency (Inbox Pattern)
	// ใช้ตรวจสอบว่า Message ID นี้เคยทำไปหรือยัง
	HasProcessedMessage(ctx context.Context, messageID string) (bool, error)
	// บันทึกว่าทำเสร็จแล้ว
	SaveProcessedMessage(ctx context.Context, messageID string) error
}

// =================================================================
// QUERY REPOSITORY (ต่อกับ Read Replica, Redis, หรือ Postgres ปกติก็ได้)
// =================================================================
type ProductQueryRepository interface {
	// ควร Return Domain หรือ DTO ก็ได้ แต่ถ้า Clean Arch เป๊ะๆ คือ Return Domain
	FindByID(ctx context.Context, id int) (*domain.Product, error)

	// ค้นหาสินค้า
	// FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error)
}
