package service

import (
	"auth_service/internal/core/domain"
	repo "auth_service/internal/core/port/repo"
	service "auth_service/internal/core/port/service"
	"context"
)

type adminService struct {
	adminRepo repo.AdminRepository
}

func NewAdminService(adminRepo *adminService) service.AdminService {
	return &adminService{
		adminRepo: adminRepo,
	}
}

func (a *adminService) CreateAdmin(ctx context.Context, user *domain.User, password string) error {
	return nil
}
func (a *adminService) UpdateAdminProfile(ctx context.Context, req *domain.User) error {
	return nil
}

func (a *adminService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	return nil
}

func (a *adminService) GetAdminProfile(ctx context.Context, id uint) (*domain.User, error) {
	return nil, nil
}
