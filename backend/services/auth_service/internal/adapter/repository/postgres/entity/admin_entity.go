package entity

import (
	"auth_service/internal/core/domain"
	gormhelper "gorm_helper"

	"gorm.io/gorm"
)

type AdminEntity struct {
	gorm.Model
	FirstName string `gorm:"type:varchar(100);not null"`
	LastName  string `gorm:"type:varchar(100);not null"`
	Username  string `gorm:"unique;not null"`
	Password  string `gorm:"type:varchar(255);not null"`
	Phone     string `gorm:"type:varchar(20)"`
	Address   string `gorm:"type:text"`
	Role      string `gorm:"type:varchar(20);default:admin;"`
}

func (a *AdminEntity) ToAdminDomain() *domain.Admin {
	if a == nil {
		return nil
	}

	gormDeletedAt := gormhelper.GormDeletedAtToTime(&a.DeletedAt)

	return &domain.Admin{
		ID:        a.ID,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
		DeletedAt: gormDeletedAt,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		Username:  a.Username,
		Password:  domain.NewPassword(a.Password),
		Phone:     a.Phone,
		Address:   a.Address,
		Role:      a.Role,
	}
}

func ToAdminEntity(admin *domain.Admin) *AdminEntity {
	if admin == nil {
		return nil
	}

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
		Password:  admin.Password.String(),
		Phone:     admin.Phone,
		Address:   admin.Address,
		Role:      admin.Role,
	}
}
