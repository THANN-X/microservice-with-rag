// WHAT: CatalogQueryService — read side ของ catalog_service (CQRS)
//
// WHY อ่านจาก MongoDB แทน PostgreSQL ของ product_service?
//   - catalog document ถูก denormalized ไว้ (product + variants + categories ใน doc เดียว)
//   - full-text search และ filter ทำได้เร็วกว่า JOIN หลาย table ใน PostgreSQL
//   - product_service sync มาผ่าน Kafka event → Eventually Consistent
//
// GetVariantInfo ใช้โดย cart_service เพื่อดึงชื่อ/ราคา/รูปสินค้า ณ เวลา checkout
package query

import (
	"catalog_service/internal/core/domain"
	repo "catalog_service/internal/core/port/repo"
	serviceport "catalog_service/internal/core/port/service"
	"catalog_service/internal/core/port/service/dto"
	"context"
	"math"
	"strconv"
	"strings"
)

type catalogQueryService struct {
	readRepo repo.CatalogReadRepository
}

func NewCatalogQueryService(readRepo repo.CatalogReadRepository) serviceport.CatalogQueryService {
	return &catalogQueryService{readRepo: readRepo}
}

func (s *catalogQueryService) SearchProducts(ctx context.Context, req *dto.SearchProductsReq) (*dto.ProductListRes, error) {
	// WHY ตั้งค่า default ใน Service Layer?
	//   - Handler เป็น thin layer: ไม่ต้องรู้ business default เรื่อง pagination
	//   - limit > 100: ป้องกัน DoS-like (ลูกค้าส่ง limit=999999 แล้ว MongoDB query จำนวนมาก)
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var categoryIDs []uint
	if req.CategoryIDs != "" {
		parts := strings.Split(req.CategoryIDs, ",")
		for _, p := range parts {
			if id, err := strconv.ParseUint(strings.TrimSpace(p), 10, 64); err == nil {
				categoryIDs = append(categoryIDs, uint(id))
			}
		}
	}

	filter := domain.ProductFilter{
		Page:        page,
		Limit:       limit,
		Search:      req.Search,
		CategoryID:  req.CategoryID,
		CategoryIDs: categoryIDs,
		SortBy:      req.SortBy,
		Order:       req.Order,
	}

	products, total, err := s.readRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CatalogProductRes, len(products))
	for i, p := range products {
		items[i] = toProductRes(p)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &dto.ProductListRes{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil
}

func (s *catalogQueryService) GetProductByID(ctx context.Context, productID uint) (*dto.CatalogProductRes, error) {
	product, err := s.readRepo.FindByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	res := toProductRes(*product)
	return &res, nil
}

// GetVariantInfo ดึงข้อมูลสั้น สำหรับแสดงใน cart (ชื่อสินค้า, ราคา, รูป)
// HOW: ค้น product doc จาก MongoDB ด้วย variantID → find variant ใน embedded array
func (s *catalogQueryService) GetVariantInfo(ctx context.Context, variantID uint) (*dto.VariantInfoRes, error) {
	product, variant, err := s.readRepo.FindByVariantID(ctx, variantID)
	if err != nil {
		return nil, err
	}

	// imageURL priority:
	//   1. variant.ImageURLs[0] — รูปเฉพาะของ variant นี้
	//   2. product.ImageURLs[0] — fallback ไปรูปหลักของ product
	//   3. "" — ถ้าไม่มีรูปเลย (cart ต้องได้รูปเสมอ)
	imageURL := ""
	if len(variant.ImageURLs) > 0 {
		imageURL = variant.ImageURLs[0]
	} else if len(product.ImageURLs) > 0 {
		imageURL = product.ImageURLs[0]
	}

	return &dto.VariantInfoRes{
		VariantID:   variant.VariantID,
		ProductID:   product.ProductID,
		ProductName: product.Name,
		VariantName: variant.Name,
		Price:       variant.Price,
		ImageURL:    imageURL,
		ImageURLs:   variant.ImageURLs,
	}, nil
}

func toProductRes(p domain.CatalogProduct) dto.CatalogProductRes {
	categories := make([]dto.CatalogCategoryRes, len(p.Categories))
	for i, c := range p.Categories {
		categories[i] = dto.CatalogCategoryRes{
			CategoryID: c.CategoryID,
			Name:       c.Name,
			Slug:       c.Slug,
		}
	}

	variants := make([]dto.CatalogVariantRes, len(p.Variants))
	for i, v := range p.Variants {
		attrs := make([]dto.VariantAttributeRes, len(v.Attributes))
		for j, a := range v.Attributes {
			attrs[j] = dto.VariantAttributeRes{Key: a.Key, Value: a.Value}
		}
		variants[i] = dto.CatalogVariantRes{
			VariantID:  v.VariantID,
			Sku:        v.Sku,
			Name:       v.Name,
			Price:      v.Price,
			Stock:      v.Stock,
			IsActive:   v.IsActive,
			ImageURLs:  v.ImageURLs,
			Attributes: attrs,
		}
	}

	return dto.CatalogProductRes{
		ProductID:   p.ProductID,
		Name:        p.Name,
		Description: p.Description,
		ImageURLs:   p.ImageURLs,
		Categories:  categories,
		Variants:    variants,
		IsActive:    p.IsActive,
	}
}
