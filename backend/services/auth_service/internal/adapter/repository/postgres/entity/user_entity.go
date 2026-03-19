package entity

import (
	"auth_service/internal/core/domain"
	gormhelper "gorm_helper"

	"gorm.io/gorm"
)

// What: UserEntity คือ GORM struct สำหรับตาราง users ใน Postgres
// Why:  แยก ORM model ออกจาก domain model เพื่อไม่ให้ domain ลายสนวััมด้วย GORM tag
//       ทำให้เปลี่ยน DB schema ได้โดยไม่กระทบ domain layer
type UserEntity struct {
	// Why: ใช้ gorm.Model เพื่อรับ ID, CreatedAt, UpdatedAt, DeletedAt อัตโนมัติ (soft-delete built-in)
	gorm.Model
	FirstName string `gorm:"type:varchar(100);not null"`
	LastName  string `gorm:"type:varchar(100);not null"`
	// Why: uniqueIndex ป้องกัน email ซ้ำที่ DB level
	Email    string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Password string `gorm:"type:varchar(255);not null"`
	Phone    string `gorm:"type:varchar(20)"`
	Address  string `gorm:"type:text"`
	// Why: default 'customer' เพื่อป้องกัน role ว่างเป็น empty string
	Role string `gorm:"type:varchar(20);default:'customer';not null"`
}

// What: แปลง UserEntity (ORM) → domain.User (pure domain)
// Why:  domain ต้องไม่เห็น GORM types (gorm.DeletedAt) — จึงแปลงที่นี่
func (u *UserEntity) ToUserDomain() *domain.User {
	if u == nil {
		return nil
	}

	// What: แปลง gorm.DeletedAt → *time.Time เพื่อให้ domain ไม่ต้อง import gorm
	deletedAt := gormhelper.GormDeletedAtToTime(&u.DeletedAt)

	return &domain.User{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		DeletedAt: deletedAt,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		// What: wrap hash string เป็น Password value object
		Password: domain.NewPassword(u.Password),
		Phone:    u.Phone,
		Address:  u.Address,
		Role:     u.Role,
	}
}

// What: แปลง domain.User → UserEntity ก่อนบันทึก DB
func ToUserEntity(user *domain.User) *UserEntity {
	if user == nil {
		return nil
	}

	// What: แปลง *time.Time → gorm.DeletedAt
	gormDeletedAt := gormhelper.TimeToGormDeletedAt(user.DeletedAt)
	return &UserEntity{
		Model: gorm.Model{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			DeletedAt: gormDeletedAt,
		},
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		// What: ดึง hash string ออกจาก Password value object เพื่อเก็บลง DB
		Password: user.Password.String(),
		Phone:    user.Phone,
		Address:  user.Address,
		Role:     user.Role,
	}
}
