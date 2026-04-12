package query

import (
	"catalog_service/internal/core/domain"
	repo "catalog_service/internal/core/port/repo"
	serviceport "catalog_service/internal/core/port/service"
	"catalog_service/internal/core/port/service/dto"
	"context"
	"math"
)

type catalogQueryService struct {
	readRepo repo.CatalogReadRepository
}

func NewCatalogQueryService(readRepo repo.CatalogReadRepository) serviceport.CatalogQueryService {
	return &catalogQueryService{readRepo: readRepo}
}

func (s *catalogQueryService) SearchProducts(ctx context.Context, req *dto.SearchProductsReq) (*dto.ProductListRes, error) {
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

	filter := domain.ProductFilter{
		Page:       page,
		Limit:      limit,
		Search:     req.Search,
		CategoryID: req.CategoryID,
		SortBy:     req.SortBy,
		Order:      req.Order,
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
