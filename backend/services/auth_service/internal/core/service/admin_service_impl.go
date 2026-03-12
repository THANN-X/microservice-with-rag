package service

import (
	"auth_service/internal/core/domain"
	repo "auth_service/internal/core/port/repo"
	service "auth_service/internal/core/port/service"
	dto "auth_service/internal/core/port/service/dto"
	"context"
	"errs"
	"logs"
)

type adminService struct {
	adminRepo repo.AdminRepository
}

func NewAdminService(adminRepo repo.AdminRepository) service.AdminService {
	return &adminService{adminRepo: adminRepo}
}

func (a *adminService) CreateAdmin(ctx context.Context, newAdminReq *dto.CreateAdminRequest, newAdminPassReq string) (*dto.AdminResponse, error) {
	newAdminDomain := &domain.Admin{
		FirstName: newAdminReq.FirstName,
		LastName:  newAdminReq.LastName,
		Username:  newAdminReq.Username,
		Phone:     newAdminReq.Phone,
		Address:   newAdminReq.Address,
	}
	// Hash the admin's password
	if err := newAdminDomain.SetPassword(newAdminPassReq); err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("password must be at least 8 characters")
	}

	err := a.adminRepo.CreateAdmin(ctx, newAdminDomain)
	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return dto.ToAdminResponse(newAdminDomain), err
}

func (a *adminService) UpdateAdminInfo(ctx context.Context, adminID uint, adminUpdateReq *dto.UpdateAdminRequest) (*domain.Admin, error) {
	return nil, nil
}

func (a *adminService) UpdatePassword(ctx context.Context, adminID uint, oldPassword, newPassword string) error {
	return nil
}

func (a *adminService) GetAdminProfile(ctx context.Context, id uint) (*domain.Admin, error) {
	return nil, nil
}
