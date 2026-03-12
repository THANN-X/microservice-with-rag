package command

import (
	"context"
	"errors"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"

	repoport "product_service/internal/core/port/repo"

	"errs"
)

type categoryCommandService struct {
	// depend เฉพาะ CommandRepository เพราะ Command Service ทำแค่ write operations
	// ไม่ต้อง inject QueryRepository ให้ overcomplicate โดยไม่จำเป็น
	cmdRepo repoport.CategoryCommandRepository
}

func NewCategoryCommandService(cmdRepo repoport.CategoryCommandRepository) port.CategoryCommandService {
	return &categoryCommandService{cmdRepo: cmdRepo}
}

// CreateCategory แปลง DTO → Domain แล้วส่งต่อให้ Repository
// ทำไมถึงสร้าง domain.Category แทนส่ง DTO ตรงๆ?
//   - Repository layer ควร depend Domain Object ไม่ใช่ DTO (Hexagonal Architecture)
//   - ทำให้ Domain Object เป็น single source of truth ของข้อมูลที่จะบันทึก
func (s *categoryCommandService) CreateCategory(ctx context.Context, req *dto.CreateCategoryReq) error {
	category := &domain.Category{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		IsActive:    req.IsActive,
		ParentID:    req.ParentID,
	}
	return s.cmdRepo.CreateCategory(ctx, category)
}

// UpdateCategory ใช้ pattern "Load → Modify → Save" (Optimistic Update)
// ทำไมต้อง Load ก่อน? เพราะ:
//  1. ถ้า Category ไม่มีอยู่ → คืน 404 ทันที (แทนที่จะ silent update 0 rows)
//  2. การ Update ทำผ่าน domain method (UpdateCategory) เพื่อให้ business rules อยู่ใน Domain
//     ไม่ใช่ Service Layer (แม้ตอนนี้ rules จะยังเบาอยู่)
func (s *categoryCommandService) UpdateCategory(ctx context.Context, req *dto.UpdateCategoryReq) error {
	// Load existing record เพื่อ validate ว่ามีอยู่จริง
	existing, err := s.cmdRepo.GetCategoryByID(ctx, req.CategoryID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError("Category not found")
		}
		return errs.NewUnexpectedError()
	}

	// Apply การเปลี่ยนแปลงผ่าน Domain method เพื่อ encapsulate update logic ไว้ใน Domain layer
	existing.UpdateCategory(&domain.Category{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		IsActive:    req.IsActive,
		ParentID:    req.ParentID,
	})

	return s.cmdRepo.UpdateCategory(ctx, existing)
}

// DeleteCategory ตรวจสอบว่า Category มีอยู่จริงก่อนลบ
// ทำไมต้อง Load ก่อนลบแทนที่จะลบตรงๆ?
//   - GORM Delete จะ succeed (ไม่ error) แม้ว่า record จะไม่มีอยู่ (RowsAffected = 0)
//   - การ Load ก่อนทำให้ Return 404 ได้อย่างชัดเจนเมื่อ Admin พยายามลบ Category ที่ไม่มีอยู่
func (s *categoryCommandService) DeleteCategory(ctx context.Context, id uint) error {
	_, err := s.cmdRepo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError("Category not found")
		}
		return errs.NewUnexpectedError()
	}
	return s.cmdRepo.DeleteCategory(ctx, id)
}

// SetCategoryActive toggle active/inactive โดยบอกสถานะผ่าน bool เดียว
// ทำไมถึงใช้ bool แทน Activate()/Deactivate()?
//   - ลด method proliferation: 1 method ทำได้ 2 หน้าที่
//   - Route เดียว (PATCH .../active) แต่ body บอกว่าต้องการอะไร
func (s *categoryCommandService) SetCategoryActive(ctx context.Context, id uint, active bool) error {
	_, err := s.cmdRepo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError("Category not found")
		}
		return errs.NewUnexpectedError()
	}
	return s.cmdRepo.SetCategoryActive(ctx, id, active)
}
