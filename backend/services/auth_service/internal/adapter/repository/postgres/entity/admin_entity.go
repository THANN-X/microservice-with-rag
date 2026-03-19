package entity

import (
	"auth_service/internal/core/domain"
	gormhelper "gorm_helper"

	"gorm.io/gorm"
)

// What: AdminEntity คือ GORM struct สำหรับตาราง admins ใน Postgres
// Why:  แยก ORM model ออกจาก domain model — pattern เดียวกับ UserEntity
type AdminEntity struct {
	gorm.Model
	FirstName string `gorm:"type:varchar(100);not null"`
	LastName  string `gorm:"type:varchar(100);not null"`
	// Why: unique เพื่อป้องกัน username ซ้ำที่ DB level
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"type:varchar(255);not null"`
	Phone     string `gorm:"type:varchar(20)"`
	Address   string `gorm:"type:text"`
	// Why: default:admin เพื่อให้แน่ใจว่า role ไม่ว่างเป็น empty string
	Role string `gorm:"type:varchar(20);default:admin;"`
}

// What: แปลง AdminEntity (ORM) → domain.Admin (pure domain)
func (a *AdminEntity) ToAdminDomain() *domain.Admin {
	if a == nil {
		return nil
	}

	// What: แปลง gorm.DeletedAt → *time.Time
	gormDeletedAt := gormhelper.GormDeletedAtToTime(&a.DeletedAt)

	return &domain.Admin{
		ID:        a.ID,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
		DeletedAt: gormDeletedAt,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		Username:  a.Username,
		// What: wrap hash string เป็น Password value object
		Password: domain.NewPassword(a.Password),
		Phone:    a.Phone,
		Address:  a.Address,
		Role:     a.Role,
	}
}

// What: แปลง domain.Admin → AdminEntity ก่อนบันทึก DB
func ToAdminEntity(admin *domain.Admin) *AdminEntity {
	if admin == nil {
		return nil
	}

	// What: แปลง *time.Time → gorm.DeletedAt
	gormDeletedAt := gormhelper.TimeToGormDeletedAt(admin.DeletedAt)

	return &AdminEntity{
		Model: gorm.Model{
			ID:        admin.ID,
			CreatedAt: admin.CreatedAt,
			UpdatedAt: admin.UpdatedAt,
			DeletedAt: gormDeletedAt,
		},
		FirstName: admin.FirstName,
		LastName:  admin.LastName,
		Username:  admin.Username,
		// What: ดึง hash string ออกจาก Password value object
		Password: admin.Password.String(),
		Phone:    admin.Phone,
		Address:  admin.Address,
		Role:     admin.Role,
	}
}
