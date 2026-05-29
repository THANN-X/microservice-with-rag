package repository

import (
	"auth_service/internal/adapter/repository/postgres/entity"
	"auth_service/internal/core/domain"
	port "auth_service/internal/core/port/repo"
	"context"
	"errors"

	"gorm.io/gorm"
)

// What: adminRepositoryDB คือ Postgres implementation ของ AdminRepository interface
type adminRepositoryDB struct {
	db *gorm.DB
}

// What: constructor — return เป็น interface เพื่อ Dependency Inversion
func NewAdminRepository(db *gorm.DB) port.AdminRepository {
	return &adminRepositoryDB{db: db}
}

// What: บันทึก admin ใหม่ลง DB แล้ว sync ID + timestamp กลับไปยัง domain object
func (r *adminRepositoryDB) CreateAdmin(ctx context.Context, admin *domain.Admin) error {
	// What: แปลง domain model → GORM entity
	adminEntity := entity.ToAdminEntity(admin)

	if err := r.db.WithContext(ctx).Create(adminEntity); err != nil {
		return err.Error
	}

	// What: sync ค่าที่ DB generate กลับไปยัง domain object
	admin.ID = adminEntity.ID
	admin.CreatedAt = adminEntity.CreatedAt
	admin.UpdatedAt = adminEntity.UpdatedAt

	return nil
}

// What: อัปเดต admin ที่มีอยู่แล้ว
// TODO: เพิ่ม handler + service route สำหรับ update admin profile
func (r *adminRepositoryDB) UpdateAdmin(ctx context.Context, admin *domain.Admin) error {
	adminEntity := entity.ToAdminEntity(admin)

	result := r.db.WithContext(ctx).Save(adminEntity)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	admin.UpdatedAt = adminEntity.UpdatedAt

	return nil
}

// What: soft-delete admin (GORM จะเติม deleted_at แทนลบจริง)
// TODO: เพิ่ม handler + service route สำหรับ delete admin
func (r *adminRepositoryDB) DeleteAdmin(ctx context.Context, id uint) error {
	adminEntity := &entity.AdminEntity{}

	result := r.db.WithContext(ctx).Delete(adminEntity, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	return nil
}

// What: ค้นหา admin ด้วย ID — ใช้ตอนดึงโปรไฟล์ผ่าน JWT claims
func (r *adminRepositoryDB) FindByID(ctx context.Context, id uint) (*domain.Admin, error) {
	adminEntity := &entity.AdminEntity{}

	result := r.db.WithContext(ctx).First(adminEntity, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return adminEntity.ToAdminDomain(), nil
}

// What: ค้นหา admin ด้วย username — ใช้ตอน admin login
// Why:  คืน nil (ไม่ใช่ error) ถ้าไม่เจอ — ต่างจาก user repo ที่คืน domain error
func (r *adminRepositoryDB) FindByUsername(ctx context.Context, username string) (*domain.Admin, error) {
	adminEntity := &entity.AdminEntity{}

	result := r.db.WithContext(ctx).First(adminEntity, "username = ?", username)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// What: คืน nil แทน error เพื่อให้ service ตรวจสอบเองว่า admin == nil
			return nil, nil
		}
		return nil, result.Error
	}

	return adminEntity.ToAdminDomain(), nil
}
