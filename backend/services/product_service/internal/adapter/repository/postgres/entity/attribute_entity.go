package entity

import (
	"product_service/internal/core/domain"

	"gorm.io/gorm"
)

type AttributeValueEntity struct {
	gorm.Model
	AttributeID uint            `gorm:"column:attribute_id;not null;uniqueIndex:idx_attr_val_unique"`
	Value       string          `gorm:"not null;uniqueIndex:idx_attr_val_unique;type:varchar(100)"`
	Attribute   AttributeEntity `gorm:"foreignKey:AttributeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type AttributeEntity struct {
	gorm.Model
	Name string `gorm:"not null;uniqueIndex;type:varchar(100)"`
}

func (e *AttributeValueEntity) ToVariantAttributeDomain() *domain.VariantAttribute {
	return &domain.VariantAttribute{
		ID:    e.ID,
		Name:  e.Attribute.Name,
		Value: e.Value,
	}
}

func ToAttributeValueEntity(d *domain.VariantAttribute) *AttributeValueEntity {
	return &AttributeValueEntity{
		Model: gorm.Model{
			ID: d.ID,
		},
		Value:     d.Value,
		Attribute: AttributeEntity{Name: d.Name},
	}
}
