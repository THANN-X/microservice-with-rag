package repository

import (
	"auth_service/internal/adapter/repository/postgres/entity"
	"auth_service/internal/core/domain"
	port "auth_service/internal/core/port/repo"
	"context"
	"errors"

	"gorm.io/gorm"
)

// What: sessionRepositoryDB คือ Postgres implementation ของ SessionRepository
// Why:  เก็บ session ไว้ใน DB เพื่อให้ revoke refresh token ได้ทันที (stateful approach)
type sessionRepositoryDB struct {
	db *gorm.DB
}

// What: constructor — return เป็น interface
func NewSessionRepository(db *gorm.DB) port.SessionRepository {
	return &sessionRepositoryDB{db: db}
}

// What: บันทึก session ใหม่ลง DB แล้ว sync ID กลับไปยัง domain object
func (r *sessionRepositoryDB) CreateSession(ctx context.Context, session *domain.Session) error {
	// What: แปลง domain → entity ก่อนบันทึก
	sessionEntity := entity.FromDomainSession(session)

	if err := r.db.WithContext(ctx).Create(sessionEntity); err != nil {
		return err.Error
	}

	// What: sync ID ที่ DB generate กลับไป
	session.ID = sessionEntity.ID

	return nil
}

// What: ค้นหา session ด้วย refresh token — ใช้ใน logout และ refresh
// Why:  token เป็น unique (unique index) จึง query แบบนี้ได้
func (r *sessionRepositoryDB) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	sessionEntity := &entity.SessionEntity{}

	result := r.db.WithContext(ctx).Where("refresh_token = ? ", refreshToken).First(&sessionEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// What: คืน domain error เพื่อให้ service ใช้ errors.Is() ได้
			return nil, domain.ErrSessionNotFound
		}
		return nil, result.Error
	}
	return sessionEntity.ToSessionDomain(), nil
}

// What: mark session ว่า revoked=true — soft invalidation ของ refresh token
// Why:  ไม่ลบ session ออก เพื่อ audit / debug ภายหลัง
// TODO: เพิ่ม cron job ลบ session ที่ revoked แล้วและ expired เกิน X วัน เพื่อไม่ให้ table ใหญ่เกิน
func (r *sessionRepositoryDB) RevokeSession(ctx context.Context, refreshToken string) error {
	return r.db.WithContext(ctx).Model(&entity.SessionEntity{}).Where("refresh_token = ?", refreshToken).Update("is_revoked", true).Error
}
