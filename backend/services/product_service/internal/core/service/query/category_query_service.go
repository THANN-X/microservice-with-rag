package query

import (
	"context"
	"errors"
	"product_service/internal/core/domain"
	repoport "product_service/internal/core/port/repo"
	port "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"

	"errs"
)

type categoryQueryService struct {
	// Query Service depend เฉพาะ QueryRepository
	// ทำให้ swap ไปใช้ Redis Cache หรือ Read Replica ได้โดยไม่กระทบ Command side
	queryRepo repoport.CategoryQueryRepository
}

func NewCategoryQueryService(queryRepo repoport.CategoryQueryRepository) port.CategoryQueryService {
	return &categoryQueryService{queryRepo: queryRepo}
}

// toCategoryRes แปลง Domain Object → Response DTO แบบ recursive
// ทำไมต้อง recursive? เพราะ Category เป็น Tree ที่มี Children ซ้อนกันหลายระดับ
// Recursion จะ map Children แต่ละตัวให้เป็น CategoryRes ด้วย
// ทำงานได้เพราะ Repository โหลด Children มาให้แล้วผ่าน GORM Preload
func toCategoryRes(d domain.Category) dto.CategoryRes {
	res := dto.CategoryRes{
		ID:          d.ID,
		Name:        d.Name,
		Slug:        d.Slug,
		Description: d.Description,
		IsActive:    d.IsActive,
		ParentID:    d.ParentID,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	// ถ้า Children ว่างเปล่า loop จะไม่ทำงาน และ res.Children จะเป็น nil
	// ซึ่ง JSON marshal เป็น omitempty → ไม่ส่ง field นี้ถ้าไม่มีลูก (ประหยัด payload)
	for _, child := range d.Children {
		res.Children = append(res.Children, toCategoryRes(child))
	}
	return res
}

// GetAllCategories คืน category tree ในรูปแบบที่ Frontend พร้อมแสดงผลได้ทันที
// ไม่ expose Domain error ออกไป แปลงเป็น HTTP-friendly error ผ่าน errs package แทน
func (s *categoryQueryService) GetAllCategories(ctx context.Context) ([]dto.CategoryRes, error) {
	categories, err := s.queryRepo.GetAllCategories(ctx)
	if err != nil {
		// ซ่อน internal error ไม่ให้รั่วไปหา Client
		return nil, errs.NewUnexpectedError()
	}

	result := make([]dto.CategoryRes, 0, len(categories))
	for _, c := range categories {
		result = append(result, toCategoryRes(c))
	}
	return result, nil
}

func (s *categoryQueryService) GetCategoryByID(ctx context.Context, id uint) (*dto.CategoryRes, error) {
	category, err := s.queryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		// errors.Is ใช้เพราะ Domain error อาจถูก wrap ผ่าน fmt.Errorf ในอนาคต
		if errors.Is(err, domain.ErrRecordNotFound) {
			return nil, errs.NewNotFoundError("Category not found")
		}
		return nil, errs.NewUnexpectedError()
	}
	res := toCategoryRes(*category)
	return &res, nil
}
