package entity

import (
	"product_service/internal/core/domain"

	"gorm.io/gorm"
)

type CategoryEntity struct {
	gorm.Model
	Name        string `gorm:"column:name"`
	Slug        string `gorm:"unique;column:slug"`
	ParentID    *uint  `gorm:"column:parent_id"`
	Description string
	IsActive    bool `gorm:"default:true"`
	// Self-referencing association
	Children []CategoryEntity `gorm:"foreignKey:ParentID"`
}

func (CategoryEntity) TableName() string {
	return "category_entities"
}

func ToCategoryEntity(d *domain.Category) *CategoryEntity {
	if d == nil {
		return nil
	}

	return &CategoryEntity{
		Model: gorm.Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		Name:        d.Name,
		Slug:        d.Slug,
		ParentID:    d.ParentID,
		Description: d.Description,
		IsActive:    d.IsActive,
	}
}

func (e *CategoryEntity) ToCategoryDomain() *domain.Category {
	if e == nil {
		return nil
	}

	d := &domain.Category{
		ID:          e.ID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Name:        e.Name,
		Slug:        e.Slug,
		ParentID:    e.ParentID,
		Description: e.Description,
		IsActive:    e.IsActive,
	}

	for _, child := range e.Children {
		d.Children = append(d.Children, *child.ToCategoryDomain())
	}

	return d
}
