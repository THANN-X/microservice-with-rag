package entity

import (
	"auth_service/internal/core/domain"
	gormhelper "gorm_helper"
	"time"

	"gorm.io/gorm"
)

// What: SessionEntity คือ GORM struct สำหรับตาราง sessions ใน Postgres
// Why:  เก็บไว้เพื่อให้ logout / token revocation ทำงานได้โดยไม่ต้องพึ่ง JWT expiry เอกเทียบ
type SessionEntity struct {
	gorm.Model
	// Why: ทั้งสองเป็น nullable pointer เพราะ session เป็นของ user หรือ admin อย่างใดอย่างหนึ่ง
	UserID  *uint        `gorm:"column:user_id;default:null"`
	// Why: cascade ลบ session อัตโนมัติเมื่อ user ถูกลบ
	User    *UserEntity  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AdminID *uint        `gorm:"column:admin_id;default:null"`
	Admin   *AdminEntity `gorm:"foreignKey:AdminID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	// Why: unique index ป้องกัน refresh token ซ้ำ และเพิ่มความเร็วในการค้นหา
	RefreshToken string    `gorm:"unique;not null;index"`
	DeviceInfo   string    `gorm:"type:varchar(255)"`
	// Why: varchar(45) รองรับทั้ง IPv4 (15) และ IPv6 (45)
	IPAddress string    `gorm:"not null;type:varchar(45)"`
	// Why: index บน ExpiredAt เพื่อ query cleanup session ที่หมดอายุได้เร็ว
	ExpiredAt time.Time `gorm:"not null;index"`
	// Why: default:false ป้องกัน null ใน column นี้
	IsRevoked bool      `gorm:"not null;default:false"`
}

// What: แปลง SessionEntity (ORM) → domain.Session
// Why:  map foreign key relations (User, Admin) กลับเป็น domain objects ด้วย
func (s *SessionEntity) ToSessionDomain() *domain.Session {
	if s == nil {
		return nil
	}

	deletedAt := gormhelper.GormDeletedAtToTime(&s.DeletedAt)

	// What: แปลง nested entity → domain เฉพาะที่มีข้อมูล (preloaded)
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

// What: แปลง domain.Session → SessionEntity ก่อนบันทึก DB
func FromDomainSession(session *domain.Session) *SessionEntity {
	if session == nil {
		return nil
	}

	gormDeletedAt := gormhelper.TimeToGormDeletedAt(session.DeletedAt)
	// What: แปลง nested domain objects → entities (ถ้ามี)
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
