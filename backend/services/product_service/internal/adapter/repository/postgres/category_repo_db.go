package repository

import (
	"context"
	"errors"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/repo"

	"gorm.io/gorm"
)

type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository คืน interface 2 ตัวจาก struct เดียวกัน (CQRS pattern)
// ทำไมถึงใช้ struct เดียวสำหรับทั้ง Command และ Query?
//   - Category ไม่ได้มีปริมาณ traffic สูงพอที่จะต้องแยก DB connection
//   - แต่ยังคง interface แยกไว้ เพื่อให้ Service Layer depend แค่สิ่งที่ตัวเองใช้
//   - ถ้าในอนาคตต้องแยก Read Replica ก็แค่ implement Query interface ใหม่โดยไม่ต้องแก้ Command
func NewCategoryRepository(db *gorm.DB) (port.CategoryCommandRepository, port.CategoryQueryRepository) {
	repo := &categoryRepository{db: db}
	return repo, repo
}

func (r *categoryRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	e := entity.ToCategoryEntity(category)
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return err
	}
	// Sync กลับ ID และ Timestamp ที่ DB generate มาให้ Domain Object
	// เพื่อให้ caller (Service Layer) รู้ว่า record ถูก save ด้วย ID อะไร
	category.ID = e.ID
	category.CreatedAt = e.CreatedAt
	category.UpdatedAt = e.UpdatedAt
	return nil
}

func (r *categoryRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	e := entity.ToCategoryEntity(category)
	// ใช้ Save แทน Updates เพราะต้องการ update ทุก field รวมถึง IsActive=false ด้วย
	// Updates จะ skip zero-value fields ทำให้ `is_active = false` ถูกโยนทิ้ง
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *categoryRepository) DeleteCategory(ctx context.Context, id uint) error {
	// GORM Soft Delete: CategoryEntity มี gorm.Model ซึ่งมี DeletedAt field
	// ดังนั้น Delete นี้จะ set deleted_at = NOW() ไม่ได้ลบจริง
	return r.db.WithContext(ctx).Delete(&entity.CategoryEntity{}, id).Error
}

func (r *categoryRepository) GetCategoryByID(ctx context.Context, id uint) (*domain.Category, error) {
	var e entity.CategoryEntity

	err := r.db.WithContext(ctx).
		// Preload Children 1 level สำหรับ GetByID (ใช้แสดง sub-categories ทันที)
		Preload("Children").
		First(&e, id).Error

	if err != nil {
		// แปลง gorm.ErrRecordNotFound → domain.ErrRecordNotFound
		// เพื่อให้ Infra error ไม่รั่วออกไปถึง Service/Handler layer
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRecordNotFound
		}
		return nil, err
	}
	return e.ToCategoryDomain(), nil
}

// SetCategoryActive ทำ targeted UPDATE is_active แทน full Save
// WHY ไม่ใช้ UpdateCategory?
//   - UpdateCategory ต้อง load object มาก่อน และแก้ทุก field → ซีด performance + risk overwrite
//   - การ toggle active ควรเป็น single-field update เสมอ (Principle of Least Privilege)
func (r *categoryRepository) SetCategoryActive(ctx context.Context, id uint, active bool) error {
	result := r.db.WithContext(ctx).
		Model(&entity.CategoryEntity{}).
		Where("id = ?", id).
		Update("is_active", active)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrRecordNotFound
	}
	return nil
}

func (r *categoryRepository) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	var entities []entity.CategoryEntity
	// WHERE parent_id IS NULL เพื่อดึงเฉพาะ Root categories
	// แล้วค่อย Preload Children ลงมา 2 ระดับ (Root → L1 → L2)
	// ทำไมไม่ดึงทุก category แล้วมา assemble ใน Go?
	//   - BFS ใน Go ต้องใช้หลาย query (เหมือน getAllSubCategoryIDs ใน product repo)
	//   - GORM Preload ทำ query แยกต่อหาก แต่ join ให้อัตโนมัติ อ่านง่ายกว่า
	//   - 2 ระดับเพียงพอสำหรับ e-commerce ทั่วไป (Electronics > Phones > Smartphones)
	err := r.db.WithContext(ctx).
		Where("parent_id IS NULL").
		Preload("Children").
		Preload("Children.Children").
		Find(&entities).Error

	if err != nil {
		return nil, err
	}

	result := make([]domain.Category, 0, len(entities))
	for _, e := range entities {
		result = append(result, *e.ToCategoryDomain())
	}
	return result, nil
}
