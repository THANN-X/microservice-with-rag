package repository

import (
	"context"
	"errors"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/repo"

	"gorm.io/gorm"
)

type attributeRepository struct {
	db *gorm.DB
}

// NewAttributeRepository คืน Command และ Query interface จาก struct เดียวกัน
// เหตุผลเดียวกับ CategoryRepository: Attribute ไม่ได้มี traffic สูงพอที่ต้องแยก DB
// แต่การแยก interface ทำให้ Service แต่ละตัว depend เฉพาะ contract ที่ตัวเองใช้จริง
func NewAttributeRepository(db *gorm.DB) (port.AttributeCommandRepository, port.AttributeQueryRepository) {
	repo := &attributeRepository{db: db}
	return repo, repo
}

func (r *attributeRepository) CreateAttribute(ctx context.Context, attr *domain.Attribute) error {
	e := entity.ToAttributeEntityFromDomain(attr)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return err
	}
	// Sync ID ที่ DB generate กลับไปยัง Domain Object
	// เพื่อให้ caller รู้ ID ของ record ที่เพิ่งสร้าง (ใช้ response หรือ log ต่อได้)
	attr.ID = e.ID
	return nil
}

func (r *attributeRepository) UpdateAttribute(ctx context.Context, attr *domain.Attribute) error {
	// ใช้ Save แทน targeted Update เพราะรับ Domain Object ทั้งก้อนมาแล้ว
	// สอดคล้องกับ DDD pattern "load whole → modify via domain method → save whole"
	// GORM จะไม่ overwrite created_at เพราะมัน autoCreateTime (set ได้เฉพาะตอน INSERT เท่านั้น)
	e := entity.ToAttributeEntityFromDomain(attr)
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *attributeRepository) DeleteAttribute(ctx context.Context, id uint) error {
	// GORM Soft Delete ผ่าน gorm.Model.DeletedAt
	// AttributeValueEntity มี OnDelete:CASCADE ใน FK จึงลบ Values ออกด้วยอัตโนมัติเมื่อ Attribute ถูกลบ
	return r.db.WithContext(ctx).Delete(&entity.AttributeEntity{}, id).Error
}

func (r *attributeRepository) CreateAttributeValue(ctx context.Context, val *domain.AttributeValue) error {
	e := entity.ToAttributeValueEntityFromDomain(val)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		// DB มี uniqueIndex:(attribute_id, value) → ถ้าเพิ่ม value ซ้ำจะได้ error ตรงนี้
		return err
	}
	val.ID = e.ID
	return nil
}

func (r *attributeRepository) DeleteAttributeValue(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.AttributeValueEntity{}, id).Error
}

// GetAllAttributes คืนเฉพาะ Attribute headers (ไม่รวม Values)
// Values จะถูกโหลดแยกโดย Query Service ผ่าน GetValuesByAttributeID
// ทำแบบนี้เพราะ:
//   - ถ้า JOIN ใน SQL เดียว GORM จะคืน rows ที่ซ้ำกัน (1 row ต่อ 1 value)
//     ต้องมา de-duplicate เองใน Go ซึ่งซับซ้อนกว่า
//   - แยก query ทำให้อ่านง่ายกว่าและ test แต่ละส่วนได้อิสระ
func (r *attributeRepository) GetAllAttributes(ctx context.Context) ([]domain.Attribute, error) {
	var entities []entity.AttributeEntity
	if err := r.db.WithContext(ctx).Find(&entities).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Attribute, 0, len(entities))
	for _, e := range entities {
		result = append(result, *e.ToAttributeDomain())
	}
	return result, nil
}

func (r *attributeRepository) GetAttributeByID(ctx context.Context, id uint) (*domain.Attribute, error) {
	var e entity.AttributeEntity
	if err := r.db.WithContext(ctx).First(&e, id).Error; err != nil {
		// แปลง GORM error → Domain error เพื่อไม่ให้ Infra layer รั่วออกไปถึง Service
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRecordNotFound
		}
		return nil, err
	}
	return e.ToAttributeDomain(), nil
}

// GetValuesByAttributeID โหลด values ทั้งหมดของ attribute ที่ระบุ
// ใช้ WHERE attribute_id = ? แทน Preload เพราะเรียกแบบ explicit จาก Service
// ทำให้ Service ควบคุมได้ว่าจะโหลด values ตอนไหน ไม่ถูก Preload โดย implicit
func (r *attributeRepository) GetValuesByAttributeID(ctx context.Context, attributeID uint) ([]domain.AttributeValue, error) {
	var entities []entity.AttributeValueEntity
	if err := r.db.WithContext(ctx).Where("attribute_id = ?", attributeID).Find(&entities).Error; err != nil {
		return nil, err
	}
	// pre-allocate slice ด้วย len ที่รู้แน่นอนแล้ว เพื่อหลีกเลี่ยง re-allocation ระหว่าง append
	result := make([]domain.AttributeValue, 0, len(entities))
	for _, e := range entities {
		result = append(result, *e.ToAttributeValueDomain())
	}
	return result, nil
}
