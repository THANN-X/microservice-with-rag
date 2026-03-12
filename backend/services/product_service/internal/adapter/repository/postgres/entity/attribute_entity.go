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
	if e == nil {
		return nil
	}

	return &domain.VariantAttribute{
		ID:    e.ID,
		Value: e.Value,
		Name:  e.Attribute.Name,
	}
}

// ใช้กับ Variant (many-to-many linking) — Domain: VariantAttribute
func ToAttributeValueEntity(d *domain.VariantAttribute) *AttributeValueEntity {
	if d == nil {
		return nil
	}

	// Map แค่ ID เป็นค่าเริ่มต้น (สำคัญที่สุดสำหรับการทำ many2many linking)
	entity := &AttributeValueEntity{
		Model: gorm.Model{
			ID: d.ID,
		},
	}

	// ถ้า Domain มี Value ส่งมาด้วย (เผื่อในอนาคตมี Use Case อื่นที่ต้องใช้) ค่อยใส่เพิ่ม
	if d.Value != "" {
		entity.Value = d.Value
	}

	// ถ้า Domain มี Name ส่งมาด้วย ค่อยสร้าง struct Attribute ซ้อนเข้าไป
	if d.Name != "" {
		entity.Attribute = AttributeEntity{
			Name: d.Name,
		}
	}

	return entity
}

// Admin-managed Attribute converters
// ใช้กับ Admin manage attribute values — Domain: AttributeValue
// ตรงไปตรงมา: map ทุก field เลย ไม่มีเงื่อนไข
// เพราะ Admin ส่งมาครบเสมอ
func ToAttributeEntityFromDomain(d *domain.Attribute) *AttributeEntity {
	if d == nil {
		return nil
	}
	return &AttributeEntity{
		Model: gorm.Model{ID: d.ID},
		Name:  d.Name,
	}
}

func (e *AttributeEntity) ToAttributeDomain() *domain.Attribute {
	if e == nil {
		return nil
	}
	return &domain.Attribute{
		ID:   e.ID,
		Name: e.Name,
	}
}

func ToAttributeValueEntityFromDomain(d *domain.AttributeValue) *AttributeValueEntity {
	if d == nil {
		return nil
	}
	return &AttributeValueEntity{
		Model:       gorm.Model{ID: d.ID},
		AttributeID: d.AttributeID,
		Value:       d.Value,
	}
}

func (e *AttributeValueEntity) ToAttributeValueDomain() *domain.AttributeValue {
	if e == nil {
		return nil
	}
	return &domain.AttributeValue{
		ID:          e.ID,
		AttributeID: e.AttributeID,
		Value:       e.Value,
	}
}
