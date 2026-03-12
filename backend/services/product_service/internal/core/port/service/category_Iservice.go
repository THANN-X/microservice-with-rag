package port

import (
	"context"
	dto "product_service/internal/core/port/service/dto"
)

type CategoryCommandService interface {
	CreateCategory(ctx context.Context, req *dto.CreateCategoryReq) error
	UpdateCategory(ctx context.Context, req *dto.UpdateCategoryReq) error
	DeleteCategory(ctx context.Context, id uint) error
	// SetCategoryActive ใช้คู่กับ UpdateCategory แต่แยกออกมา เพราะ:
	//   - เป็น operation ที่ใช้บ่อยและไม่ควรจำเป็นต้องส่ง body ทั้งหมดเหมือน UpdateCategory
	//   - ทำให้ route ชัดเจน: PATCH /categories/admin/:id/active
	SetCategoryActive(ctx context.Context, id uint, active bool) error
}

type CategoryQueryService interface {
	GetAllCategories(ctx context.Context) ([]dto.CategoryRes, error)
	GetCategoryByID(ctx context.Context, id uint) (*dto.CategoryRes, error)
}
