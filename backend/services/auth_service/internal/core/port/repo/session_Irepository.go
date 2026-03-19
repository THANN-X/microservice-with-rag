package port

import (
	"auth_service/internal/core/domain"
	"context"
)

// What: SessionRepository คือ interface สำหรับจัดการ session lifecycle
// Why:  แยก session storage ออกเป็น interface เพื่อเปลี่ยนไปใช้ Redis หรือ in-memory ได้ในอนาคต
// TODO: เพิ่ม ListSessionsByUser สำหรับดู active sessions ทั้งหมดของ user (งาน security dashboard)
type SessionRepository interface {
	// What: สร้าง session ใหม่หลัง login สำเร็จ
	CreateSession(ctx context.Context, session *domain.Session) error
	// What: ค้นหา session ด้วย refresh token — ใช้ logout/refresh
	GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error)
	// What: mark session ว่า revoked=true — ใช้ logout
	RevokeSession(ctx context.Context, refreshToken string) error
}
