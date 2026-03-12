package port

import (
	dto "auth_service/internal/core/port/service/dto"
	"context"
)

type AuthService interface {
	LoginUser(ctx context.Context, email, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error)
	LoginAdmin(ctx context.Context, username, password, ipAddress, deviceInfo string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*dto.LoginResponse, error)
}
