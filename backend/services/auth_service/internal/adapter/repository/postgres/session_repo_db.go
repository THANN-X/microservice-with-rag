package repository

import (
	"auth_service/internal/adapter/repository/postgres/entity"
	"auth_service/internal/core/domain"
	port "auth_service/internal/core/port/repo"
	"context"
	"errors"

	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) port.SessionRepository {
	return &sessionRepository{db: db}
}

// CreateSession บันทึก Session ใหม่ลง DB
func (r *sessionRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	sessionEntity := entity.FromDomainSession(session)

	if err := r.db.WithContext(ctx).Create(sessionEntity); err != nil {
		return err.Error
	}

	session.ID = sessionEntity.ID

	return nil
}
func (r *sessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	sessionEntity := &entity.SessionEntity{}

	// ค้นหา Token และต้องยังไม่ Revoked (Optionally check Revoked here or in Service)
	result := r.db.WithContext(ctx).Where("refresh_token = ? ", refreshToken).First(&sessionEntity)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, result.Error
	}
	return sessionEntity.ToSessionDomain(), nil
}

// RevokeSession ยกเลิกการใช้งาน Token (เช่น user กด Logout)
func (r *sessionRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	return r.db.WithContext(ctx).Model(&entity.SessionEntity{}).Where("refresh_token = ?", refreshToken).Update("is_revoked", true).Error
}
