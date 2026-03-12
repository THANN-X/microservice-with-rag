package port

import (
	"auth_service/internal/core/domain"
	dto "auth_service/internal/core/port/service/dto"
	"context"
)

type AdminService interface {

	// User Management
	CreateAdmin(ctx context.Context, newAdminReq *dto.CreateAdminRequest, newAdminPassReq string) (*dto.AdminResponse, error)
	UpdateAdminInfo(ctx context.Context, adminID uint, adminUpdateReq *dto.UpdateAdminRequest) (*domain.Admin, error)
	UpdatePassword(ctx context.Context, adminID uint, oldPassword, newPassword string) error

	// Authentication
	GetAdminProfile(ctx context.Context, id uint) (*domain.Admin, error)
}
