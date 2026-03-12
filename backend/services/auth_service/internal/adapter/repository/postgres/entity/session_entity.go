package entity

import (
	"auth_service/internal/core/domain"
	gormhelper "gorm_helper"
	"time"

	"gorm.io/gorm"
)

type SessionEntity struct {
	gorm.Model
	UserID       *uint        `gorm:"column:user_id;default:null"`
	User         *UserEntity  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AdminID      *uint        `gorm:"column:admin_id;default:null"`
	Admin        *AdminEntity `gorm:"foreignKey:AdminID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	RefreshToken string       `gorm:"unique;not null;index"`
	DeviceInfo   string       `gorm:"type:varchar(255)"`
	IPAddress    string       `gorm:"not null;type:varchar(45)"`
	ExpiredAt    time.Time    `gorm:"not null;index"`
	IsRevoked    bool         `gorm:"not null;default:false"`
}

func (s *SessionEntity) ToSessionDomain() *domain.Session {
	if s == nil {
		return nil
	}

	deletedAt := gormhelper.GormDeletedAtToTime(&s.DeletedAt)
	var userDomain *domain.User
	var adminDomain *domain.Admin

	if s.User != nil {
		userDomain = s.User.ToUserDomain()
	}
	if s.Admin != nil {
		adminDomain = s.Admin.ToAdminDomain()
	}

	return &domain.Session{
		ID:           s.ID,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
		DeletedAt:    deletedAt,
		UserID:       s.UserID,
		User:         userDomain,
		AdminID:      s.AdminID,
		Admin:        adminDomain,
		RefreshToken: s.RefreshToken,
		DeviceInfo:   s.DeviceInfo,
		IPAddress:    s.IPAddress,
		ExpiredAt:    s.ExpiredAt,
		IsRevoked:    s.IsRevoked,
	}
}

func FromDomainSession(session *domain.Session) *SessionEntity {
	if session == nil {
		return nil
	}

	gormDeletedAt := gormhelper.TimeToGormDeletedAt(session.DeletedAt)
	userEntity := ToUserEntity(session.User)
	adminEntity := ToAdminEntity(session.Admin)

	return &SessionEntity{
		Model: gorm.Model{
			ID:        session.ID,
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
			DeletedAt: gormDeletedAt,
		},
		UserID:       session.UserID,
		User:         userEntity,
		AdminID:      session.AdminID,
		Admin:        adminEntity,
		RefreshToken: session.RefreshToken,
		DeviceInfo:   session.DeviceInfo,
		IPAddress:    session.IPAddress,
		ExpiredAt:    session.ExpiredAt,
		IsRevoked:    session.IsRevoked,
	}
}
