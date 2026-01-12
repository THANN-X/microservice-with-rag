package port

import (
	res "auth_service/internal/core/port/service/dto"
	"context"
)

type AuthService interface {
	LoginUser(ctx context.Context, email, password, ipAddress, deviceInfo string) (*res.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*res.LoginResponse, error)
	// LoginAdmin(ctx context.Context, username, password string) (string, error)
}
