package domain

import (
	dto "product_service/internal/core/port/service/dto"
	"time"
)

type Product struct {
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	Name        string
	Description string
	Variants    []ProductVariant
	CateGories  []CateGory
}

type ProductVariant struct {
	ID          uint
	ProductID   uint
	Sku         string
	NameVariant string
	Price       float64
	Stock       int
	IsActive    bool
	Attributes  []VariantAttribute
}

func (p *Product) ToProductRes() *dto.ProductRes {
	categories := make([]string, len(p.CateGories))
	variant := make([]dto.ProductVariantRes, len(p.CateGories))

	for i, c := range p.CateGories {
		categories[i] = c.Name
	}

	for i, v := range p.Variants {
		option := make([]dto.VariantOptionRes, len(p.CateGories))

		for j, attr := range v.Attributes {
			option[j] = dto.VariantOptionRes{
				Name:  attr.Name,
				Value: attr.Value,
			}
		}

		variant[i] = dto.ProductVariantRes{
			ID:      v.ID,
			Sku:     v.Sku,
			Price:   v.Price,
			Stock:   v.Stock,
			Options: option,
		}
	}

	return &dto.ProductRes{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Variants:    variant,
		Categories:  categories,
	}
}
