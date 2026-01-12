package entity

import (
	gormhelper "gorm_helper"
	"product_service/internal/core/domain"

	"gorm.io/gorm"
)

type ProductEntity struct {
	gorm.Model
	Name        string                 `gorm:"not null;index"`
	Description string                 `gorm:"type:text"`
	Variant     []ProductVariantEntity `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CateGories  []CateGoryEntity       `gorm:"many2many:product_categories;joinForeignKey:product_id;joinReferences:attribute_id"`
}

type ProductVariantEntity struct {
	gorm.Model
	ProductID       uint                   `gorm:"column:product_id;index;not null"`
	Sku             string                 `gorm:"uniqueIndex;not null;type:varchar(100)"`
	NameVariant     string                 `gorm:"not null"`
	Price           float64                `gorm:"not null;type:decimal(10,2);default:0"`
	Stock           int                    `gorm:"not null;default:0"`
	IsActive        bool                   `gorm:"default:true"`
	AttributeValues []AttributeValueEntity `gorm:"many2many:variant_values;joinForeignKey:variant_id;joinReferences:value_id"`
}

func (e *ProductEntity) ToProductDomain() *domain.Product {
	if e == nil {
		return nil
	}

	deletedAt := gormhelper.GormDeletedAtToTime(&e.DeletedAt)

	categories := make([]domain.CateGory, len(e.CateGories))
	variants := make([]domain.ProductVariant, len(e.Variant))

	for i, c := range e.CateGories {
		categories[i] = *c.ToCateGoryDomain()
	}

	for j, v := range e.Variant {
		variants[j] = *v.ToProductVariantDomain()
	}

	return &domain.Product{
		ID:          e.ID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   deletedAt,
		Name:        e.Name,
		Description: e.Description,
		Variants:    variants,
	}
}

func ToProductEntity(d *domain.Product) *ProductEntity {
	if d == nil {
		return nil
	}

	deleteAt := gormhelper.TimeToGormDeletedAt(d.DeletedAt)

	categories := make([]CateGoryEntity, len(d.CateGories))
	variants := make([]ProductVariantEntity, len(d.Variants))

	for i, c := range d.CateGories {
		categories[i] = *ToCateGoryEntity(&c)
	}
	for j, v := range d.Variants {
		variants[j] = *ToProductVariantEntity(&v)

	}
	return &ProductEntity{
		Model: gorm.Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: deleteAt,
		},
		Name:        d.Name,
		Description: d.Description,
		Variant:     variants,
		CateGories:  categories,
	}
}

func (e *ProductVariantEntity) ToProductVariantDomain() *domain.ProductVariant {
	if e == nil {
		return nil
	}

	attributes := make([]domain.VariantAttribute, len(e.AttributeValues))

	for i, v := range e.AttributeValues {
		attributes[i] = *v.ToVariantAttributeDomain()
	}

	return &domain.ProductVariant{
		ID:          e.ID,
		ProductID:   e.ProductID,
		Sku:         e.Sku,
		NameVariant: e.NameVariant,
		Price:       e.Price,
		Stock:       e.Stock,
		IsActive:    e.IsActive,
		Attributes:  attributes,
	}
}

func ToProductVariantEntity(d *domain.ProductVariant) *ProductVariantEntity {
	if d == nil {
		return nil
	}

	attributes := make([]AttributeValueEntity, len(d.Attributes))

	for i, v := range d.Attributes {
		attributes[i] = *ToAttributeValueEntity(&v)
	}

	return &ProductVariantEntity{
		Model: gorm.Model{
			ID: d.ID,
		},
		ProductID:       d.ProductID,
		Sku:             d.Sku,
		NameVariant:     d.NameVariant,
		Price:           d.Price,
		Stock:           d.Stock,
		IsActive:        d.IsActive,
		AttributeValues: attributes,
	}
}
