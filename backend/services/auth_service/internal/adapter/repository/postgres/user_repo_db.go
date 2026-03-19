package repository

import (
	"context"
	"errors"

	"auth_service/internal/adapter/repository/postgres/entity"
	"auth_service/internal/core/domain"
	port "auth_service/internal/core/port/repo"

	"gorm.io/gorm"
)

// What: userRepositoryDB คือ Postgres implementation ของ UserRepository interface
// Why:  แยก DB logic ออกจาก domain/service ทำให้เรียกใช้ผ่าน interface และเปลี่ยน DB ได้โดยไม่กระทบโค้ด business
type userRepositoryDB struct {
	db *gorm.DB
}

// What: constructor — return เป็น interface เพื่อ enforce Dependency Inversion
func NewUserRepository(db *gorm.DB) port.UserRepository {
	return &userRepositoryDB{db: db}
}

// What: บันทึก user ใหม่ลง DB แล้ว sync ID กลับไปยัง domain object
// Why:  domain object ต้องรู้ ID ที่ DB generate ไว้เพื่อใช้ในการ create session ต่อไป
func (r *userRepositoryDB) CreateUser(ctx context.Context, user *domain.User) error {
	// What: แปลง domain model → GORM entity ก่อนบันทึก — domain ต้องไม่รู้จัก ORM
	userEntity := entity.ToUserEntity(user)

	if err := r.db.WithContext(ctx).Create(userEntity); err != nil {
		return err.Error
	}
	// What: sync ค่าที่ DB generate กลับไปยัง domain object
	user.ID = userEntity.ID
	user.CreatedAt = userEntity.CreatedAt
	user.UpdatedAt = userEntity.UpdatedAt

	return nil
}

// What: อัปเดต user ที่มีอยู่แล้วใน DB (save ทั้งแถว)
// Why:  GORM Save ทำ UPDATE ถ้ามี ID — safe สำหรับ update แบบ full-replace
// TODO: พิจารณาใช้ Updates แทน Save ถ้าต้องการ partial update เพื่อป้องกัน overwrite field ที่ไม่เป็น zero value
func (r *userRepositoryDB) UpdateUser(ctx context.Context, user *domain.User) error {
	userEntity := entity.ToUserEntity(user)
	result := r.db.WithContext(ctx).Save(userEntity)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	// What: sync UpdatedAt กลับไปยัง domain object
	user.UpdatedAt = userEntity.UpdatedAt

	return nil
}

// What: soft-delete user (GORM เติม deleted_at แทนลบจริง)
func (r *userRepositoryDB) DeleteUser(ctx context.Context, id uint) error {
	userEntity := &entity.UserEntity{}
	result := r.db.WithContext(ctx).Delete(userEntity, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	return nil
}

// What: ค้นหา user ด้วย primary key
// Why:  ใช้ First() แทน Find() เพราะเราต้องการ exactly 1 record และให้ ErrRecordNotFound ถ้าไม่เจอ
func (r *userRepositoryDB) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	userEntity := &entity.UserEntity{}
	result := r.db.WithContext(ctx).First(userEntity, id)
	if result.Error != nil {
		// What: แปลง GORM not-found error → domain error เพื่อ decouple DB จาก service
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, result.Error
	}
	return userEntity.ToUserDomain(), nil
}

// What: ค้นหา user ด้วย email (ใช้ตอน login และตรวจ email ซ้ำ)
func (r *userRepositoryDB) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	userEntity := &entity.UserEntity{}
	result := r.db.WithContext(ctx).First(userEntity, "email = ?", email)
	if result.Error != nil {
		// What: คืน domain.ErrUserNotFound เพื่อให้ service ใช้ errors.Is() ได้ตรง
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, result.Error
	}

	return userEntity.ToUserDomain(), nil
}
