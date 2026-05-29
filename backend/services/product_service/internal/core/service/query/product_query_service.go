package query

import (
	"context"
	"errors"
	"errs"
	"logs"
	"math"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/repo"
	service "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"
	"product_service/internal/core/port/service/mapper"
)

type productQueryService struct {
	queryRepo port.ProductQueryRepository
}

func NewProductQueryService(queryRepo port.ProductQueryRepository) service.ProductQueryService {
	return &productQueryService{
		queryRepo: queryRepo,
	}
}

func (s *productQueryService) GetProductByID(ctx context.Context, id uint) (*dto.ProductRes, error) {
	product, err := s.queryRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return nil, errs.NewNotFoundError("product not found")
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return mapper.ToProductRes(product), nil
}

func (s *productQueryService) ListProducts(ctx context.Context, req *dto.ListProductReq) (*dto.ProductListRes, error) {
	filter := domain.ProductFilter{
		Page:       req.Page,
		Limit:      req.Limit,
		Search:     req.Search,
		CategoryID: req.Category,
		IsActive:   req.IsActive,
		SortBy:     req.SortBy,
		Order:      req.Order,
	}

	// WHY ตั้งค่า default ใน Service แทนให้เป็น zero value?
	//   - ทำให้ควบคุม business default logic ไว้ใน Core Layer ไม่ผ่าน Handler
	//   - Handler ไม่ต้องรู้ว่าค่า default คืออะไร → thin handler
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// WHY คุม limit ไม่เกิน 100 รายการ?
	//   - ป้องกันผู้ใช้ ส่ง limit=999999 แล้วดึงข้อมูลทั้งหมด (DoS-like attack)
	//   - 100 records/page พอสำหรับ admin backoffice (user experience + performance balance)
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}

	// default sort: สินค้าใหม่สุดขึ้นก่อนเสมอ (admin backoffice expectation)
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}

	if filter.Order == "" {
		filter.Order = "desc"
	}

	products, total, err := s.queryRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}

	items := make([]dto.ProductRes, len(products))
	for i := range products {
		items[i] = *mapper.ToProductRes(&products[i])
	}

	// Total pages = ceil(total / limit)
	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	// Result shape:
	//   ProductListRes{
	//     Items:      []ProductRes  ← สินค้าในหน้านี้
	//     Total:      int64         ← จำนวนทั้งหมดที่ตรงกับ filter (ไม่ใช่แค่หน้านี้)
	//     Page:       int           ← หน้าปัจจุบัน (เริ่มที่ 1)
	//     PageSize:   int           ← จำนวนต่อหน้าที่ใช้จริง
	//     TotalPages: int           ← ceil(Total / PageSize)
	//   }
	// ตัวอย่าง: Total=25, Page=2, PageSize=10 → TotalPages=3, Items มี 10 รายการ

	return &dto.ProductListRes{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.Limit,
		TotalPages: totalPages,
	}, nil
}
