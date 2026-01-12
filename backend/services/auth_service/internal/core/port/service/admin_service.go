package port

import (
	"auth_service/internal/core/domain"
	"context"
)

type AdminService interface {

	// User Management
	CreateAdmin(ctx context.Context, user *domain.User, password string) error
	UpdateAdminProfile(ctx context.Context, req *domain.User) error
	ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error

	// Authentication
	GetAdminProfile(ctx context.Context, id uint) (*domain.User, error)
}
