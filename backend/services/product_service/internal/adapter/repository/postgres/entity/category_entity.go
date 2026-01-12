package entity

import (
	"product_service/internal/core/domain"

	"gorm.io/gorm"
)

type CateGoryEntity struct {
	gorm.Model
	Name        string `gorm:"column:name"`
	Slug        string `gorm:"unique;column:slug"`
	ParentID    *uint  `gorm:"column:parent_id"`
	Description string
	IsActive    bool             `gorm:"default:true"`
	Children    []CateGoryEntity `gorm:"foreignKey:ParentID"`
}

func ToCateGoryEntity(d *domain.CateGory) *CateGoryEntity {
	return &CateGoryEntity{
		Model: gorm.Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		Slug:        d.Slug,
		Description: d.Description,
		IsActive:    d.IsActive,
	}
}

func (e *CateGoryEntity) ToCateGoryDomain() *domain.CateGory {
	return &domain.CateGory{
		ID:          e.ID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Name:        e.Name,
		Slug:        e.Slug,
		Description: e.Description,
		IsActive:    e.IsActive,
	}
}
