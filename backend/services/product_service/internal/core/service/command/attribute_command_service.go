package command

import (
	"context"
	"errors"
	"product_service/internal/core/domain"
	repoport "product_service/internal/core/port/repo"
	port "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"

	"errs"
)

type attributeCommandService struct {
	cmdRepo repoport.AttributeCommandRepository
	// ทำไม Command Service ต้องมี QueryRepository ด้วย?
	//   - ต้องการ GetAttributeByID เพื่อ validate ว่า Attribute มีอยู่จริงก่อน Update/Delete
	//   - แทนที่จะ duplicate method นี้ใน CommandRepository ก็ใช้ QueryRepository ที่มีอยู่แล้วแทน
	//   - Query ใน step นี้เป็น read-only ไม่มี side effect จึงปลอดภัยที่ Command Service จะใช้
	queryRepo repoport.AttributeQueryRepository
}

func NewAttributeCommandService(cmdRepo repoport.AttributeCommandRepository, queryRepo repoport.AttributeQueryRepository) port.AttributeCommandService {
	return &attributeCommandService{cmdRepo: cmdRepo, queryRepo: queryRepo}
}

func (s *attributeCommandService) CreateAttribute(ctx context.Context, req *dto.CreateAttributeReq) error {
	attr := &domain.Attribute{Name: req.Name}
	return s.cmdRepo.CreateAttribute(ctx, attr)
}

// UpdateAttribute ใช้ pattern "Load → Modify → Save" เหมือนกับ Category และ Product
// ทำไมต้อง Load ก่อน?
//  1. ได้ 404 ทันทีถ้า Attribute ไม่มีอยู่ (แทน silent 0 rows)
//  2. การ Modify ผ่าน domain method (existing.Update) เพื่อ enforce business rules ที่ Domain layer
//     ป้องกักันการ set field ตรงๆ ข้ามหัว domain ใน Service
func (s *attributeCommandService) UpdateAttribute(ctx context.Context, req *dto.UpdateAttributeReq) error {
	existing, err := s.queryRepo.GetAttributeByID(ctx, req.AttributeID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError("Attribute not found")
		}
		return errs.NewUnexpectedError()
	}
	// Modify ผ่าน domain method — ไม่ set field โดยตรง
	existing.Update(req.Name)
	return s.cmdRepo.UpdateAttribute(ctx, existing)
}

func (s *attributeCommandService) DeleteAttribute(ctx context.Context, id uint) error {
	_, err := s.queryRepo.GetAttributeByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError("Attribute not found")
		}
		return errs.NewUnexpectedError()
	}
	return s.cmdRepo.DeleteAttribute(ctx, id)
}

func (s *attributeCommandService) CreateAttributeValue(ctx context.Context, req *dto.CreateAttributeValueReq) error {
	_, err := s.queryRepo.GetAttributeByID(ctx, req.AttributeID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError("Attribute not found")
		}
		return errs.NewUnexpectedError()
	}
	val := &domain.AttributeValue{
		AttributeID: req.AttributeID,
		Value:       req.Value,
	}
	return s.cmdRepo.CreateAttributeValue(ctx, val)
}

func (s *attributeCommandService) DeleteAttributeValue(ctx context.Context, id uint) error {
	return s.cmdRepo.DeleteAttributeValue(ctx, id)
}
