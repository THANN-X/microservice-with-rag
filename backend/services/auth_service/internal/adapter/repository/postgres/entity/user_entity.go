package entity

import (
	"auth_service/internal/core/domain"
	gormhelper "gorm_helper"

	"gorm.io/gorm"
)

// UserEntity represents the user table in the database.
type UserEntity struct {
	gorm.Model
	FirstName string `gorm:"type:varchar(100);not null"`
	LastName  string `gorm:"type:varchar(100);not null"`
	Email     string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Password  string `gorm:"type:varchar(255);not null"`
	Phone     string `gorm:"type:varchar(20)"`
	Address   string `gorm:"type:text"`
	Role      string `gorm:"type:varchar(20);default:'customer';not null"`
}

// ToDomain maps UserEntity to domain.User
func (u *UserEntity) ToUserDomain() *domain.User {
	deletedAt := gormhelper.GormDeletedAtToTime(&u.DeletedAt)
	return &domain.User{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		DeletedAt: deletedAt,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Password:  domain.NewPassword(u.Password),
		Phone:     u.Phone,
		Address:   u.Address,
		Role:      u.Role,
	}
}

// FromDomain maps domain.User to UserEntity
func ToUserEntity(user *domain.User) *UserEntity {
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
		Password:  user.Password.String(),
		Phone:     user.Phone,
		Address:   user.Address,
		Role:      user.Role,
	}
}
