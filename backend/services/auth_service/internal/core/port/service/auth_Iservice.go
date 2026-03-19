package port

import (
	dto "auth_service/internal/core/port/service/dto"
	"context"
)

// What: AuthService คือ interface สำหรับ login, logout และ token lifecycle
// Why:  แยกออกจาก UserService/AdminService เพราะเกี่ยวกับ authentication flow
//       ซึ่ง cross-cutting ระหว่าง user และ admin
// TODO: เพิ่ม GetActiveSessions เพื่อให้ user ดู และ revoke เฝฟ็ก session
type AuthService interface {
	LoginUser(ctx context.Context, email, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error)
	LoginAdmin(ctx context.Context, username, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*dto.LoginResponse, error)
}
