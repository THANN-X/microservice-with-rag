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

type attributeQueryService struct {
	queryRepo repoport.AttributeQueryRepository
}

func NewAttributeQueryService(queryRepo repoport.AttributeQueryRepository) port.AttributeQueryService {
	return &attributeQueryService{queryRepo: queryRepo}
}

// GetAllAttributes โหลด attributes ทั้งหมดพร้อม values โดยใช้ 2 queries แยกกัน
// ทำไมไม่ใช้ SQL JOIN แบบเดียว?
//   - JOIN จะคืน N rows ต่อ attribute (1 row ต่อ value) ต้องมา group ใน Go เอง
//   - แยก query ทำให้โค้ดอ่านง่ายและ predictable กว่า
//
// Trade-off: N+1 query (1 สำหรับ attrs + N สำหรับ values ของแต่ละ attr)
// ยอมรับได้เพราะ Attribute catalog มักมีจำนวนไม่มาก (< 100 records)
func (s *attributeQueryService) GetAllAttributes(ctx context.Context) ([]dto.AttributeRes, error) {
	attrs, err := s.queryRepo.GetAllAttributes(ctx)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}

	result := make([]dto.AttributeRes, 0, len(attrs))
	for _, a := range attrs {
		// โหลด values ของแต่ละ attribute แยกต่าง
		values, err := s.queryRepo.GetValuesByAttributeID(ctx, a.ID)
		if err != nil {
			return nil, errs.NewUnexpectedError()
		}
		res := dto.AttributeRes{
			ID:     a.ID,
			Name:   a.Name,
			Values: make([]dto.AttributeValueRes, 0),
		}
		for _, v := range values {
			res.Values = append(res.Values, dto.AttributeValueRes{
				ID:    v.ID,
				Value: v.Value,
			})
		}
		result = append(result, res)
	}
	return result, nil
}

// GetAttributeByID โหลด attribute เดียวพร้อม values ทั้งหมดของมัน
// ใช้สำหรับ Admin ดู detail ของ attribute ก่อนแก้ไข
func (s *attributeQueryService) GetAttributeByID(ctx context.Context, id uint) (*dto.AttributeRes, error) {
	attr, err := s.queryRepo.GetAttributeByID(ctx, id)
	if err != nil {
		// errors.Is รองรับกรณีที่ error ถูก wrap ด้วย fmt.Errorf("%w", err) ในอนาคต
		if errors.Is(err, domain.ErrRecordNotFound) {
			return nil, errs.NewNotFoundError("Attribute not found")
		}
		return nil, errs.NewUnexpectedError()
	}

	values, err := s.queryRepo.GetValuesByAttributeID(ctx, id)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}

	res := &dto.AttributeRes{
		ID:     attr.ID,
		Name:   attr.Name,
		Values: make([]dto.AttributeValueRes, 0),
	}
	for _, v := range values {
		res.Values = append(res.Values, dto.AttributeValueRes{
			ID:    v.ID,
			Value: v.Value,
		})
	}
	return res, nil
}
