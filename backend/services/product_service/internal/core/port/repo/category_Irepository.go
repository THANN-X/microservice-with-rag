package port

import (
	"context"
	"product_service/internal/core/domain"
)

// CategoryCommandRepository กำหนด contract ของ Write operations
// เหตุที่ GetCategoryByID อยู่ที่นี่ด้วย (ไม่ใช่แค่ Query):
//   - Command Service (UpdateCategory, DeleteCategory) ต้องโหลด record เพื่อ validate ก่อนแก้ไข
//   - แยก interface ออกจาก Query เพื่อให้ Command Service ไม่ต้อง depend Query Repository
type CategoryCommandRepository interface {
	CreateCategory(ctx context.Context, category *domain.Category) error
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id uint) error
	// GetCategoryByID ใช้สำหรับ lookup ก่อน Update/Delete เพื่อ return 404 แทน silent fail
	GetCategoryByID(ctx context.Context, id uint) (*domain.Category, error)
	// SetCategoryActive ทำ targeted UPDATE is_active ไม่ลาก field อื่น
	SetCategoryActive(ctx context.Context, id uint, active bool) error
}

// CategoryQueryRepository กำหนด contract ของ Read operations
// ถูกออกแบบให้รองรับ CQRS: Query side อาจชี้ไปที่ Read Replica หรือ Cache ในอนาคต
// โดยไม่กระทบ Command side
type CategoryQueryRepository interface {
	// GetAllCategories คืน Root categories พร้อม Children (Tree structure)
	GetAllCategories(ctx context.Context) ([]domain.Category, error)
	GetCategoryByID(ctx context.Context, id uint) (*domain.Category, error)
}
