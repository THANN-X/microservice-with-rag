package entity

import (
	gormhelper "gorm_helper"
	"product_service/internal/core/domain"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ProductEntity คือ GORM model ที่แมปสามารถเชื่อมต่อกับตาราง products ใน DB
// WHY แยก Entity ออกจาก Domain?
//   - Domain Object ไม่ควรรู้จัก GORM tag หรือโครงสร้าง DB (Clean Architecture)
//   - ถ้าไม่แยก GORM จะไปรั่ว domain model นี้ ส่งผลให้ Unit Test ยาก (ต้อง mock DB)
type ProductEntity struct {
	gorm.Model
	Name        string                 `gorm:"not null;index"`
	Description string                 `gorm:"type:text"`
	ImageURLs   pq.StringArray         `gorm:"type:text[]"`
	Variants    []ProductVariantEntity `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Categories  []CategoryEntity       `gorm:"many2many:product_categories;joinForeignKey:product_id;joinReferences:category_id"`
	// IsActive แยกจาก DeletedAt (soft delete) เพราะ semantic ต่างกัน:
	// - DeletedAt = ลบแล้ว ไม่ควรปรากฏทั้ง admin และ frontend
	// - IsActive = admin ยังเห็น แต่ customer ไม่เห็น (draft/unpublished)
	IsActive  bool `gorm:"default:true"`
	CreatedBy uint `gorm:"not null;index"` // Index ไว้เผื่อ search สินค้าตามคนสร้าง
	UpdatedBy uint `gorm:"not null"`
}

type ProductVariantEntity struct {
	gorm.Model
	ProductID       uint                   `gorm:"column:product_id;index;not null"`
	Sku             string                 `gorm:"uniqueIndex;not null;type:varchar(100)"`
	NameVariant     string                 `gorm:"not null"`
	Price           float64                `gorm:"not null;type:decimal(10,2);default:0"`
	Stock           int                    `gorm:"not null;default:0"`
	IsActive        bool                   `gorm:"default:true"`
	ImageURLs       pq.StringArray         `gorm:"type:text[]"`
	AttributeValues []AttributeValueEntity `gorm:"many2many:variant_values;joinForeignKey:variant_id;joinReferences:value_id"`
}

// ToProductDomain แปลง GORM entity เป็น Domain Object (Anti-Corruption Layer)
// WHY nil check?
//   - GORM Preload บางครั้งคืน zero-value struct แทน nil → ป้องกัน nil pointer dereference
func (e *ProductEntity) ToProductDomain() *domain.Product {
	if e == nil {
		return nil
	}

	deletedAt := gormhelper.GormDeletedAtToTime(&e.DeletedAt)

	categories := make([]domain.Category, len(e.Categories))
	variants := make([]domain.ProductVariant, len(e.Variants))

	for i, c := range e.Categories {
		categories[i] = *c.ToCategoryDomain()
	}

	for j, v := range e.Variants {
		variants[j] = *v.ToProductVariantDomain()
	}

	return &domain.Product{
		ID:          e.ID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   deletedAt,
		Name:        e.Name,
		Description: e.Description,
		ImageURLs:   []string(e.ImageURLs),
		Categories:  categories,
		Variants:    variants,
		IsActive:    e.IsActive,
		CreatedBy:   e.CreatedBy,
		UpdatedBy:   e.UpdatedBy,
	}
}

// ToProductEntity แปลง Domain Object เป็น GORM entity ก่อน save ลง DB
// WHY เซ็ต gorm.Model.ID ด้วย?
//   - ถ้า ID = 0 GORM จะ INSERT ถ้า ID > 0 GORM จะ UPDATE
//   - ทำให้สามารถใช้ function เดียวกันได้ทั้ง Create และ Update
func ToProductEntity(d *domain.Product) *ProductEntity {
	if d == nil {
		return nil
	}

	deleteAt := gormhelper.TimeToGormDeletedAt(d.DeletedAt)

	categories := make([]CategoryEntity, len(d.Categories))
	variants := make([]ProductVariantEntity, len(d.Variants))

	for i, c := range d.Categories {
		categories[i] = *ToCategoryEntity(&c)
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
		ImageURLs:   pq.StringArray(d.ImageURLs),
		Variants:    variants,
		Categories:  categories,
		IsActive:    d.IsActive,
		CreatedBy:   d.CreatedBy,
		UpdatedBy:   d.UpdatedBy,
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
		ImageURLs:   []string(e.ImageURLs),
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
		ImageURLs:       pq.StringArray(d.ImageURLs),
		AttributeValues: attributes,
	}
}
