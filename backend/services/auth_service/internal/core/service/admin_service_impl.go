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

// What: adminService จัดการ business logic เกี่ยวกับ admin account
type adminService struct {
	adminRepo repo.AdminRepository
}

// What: constructor — inject repository
func NewAdminService(adminRepo repo.AdminRepository) service.AdminService {
	return &adminService{adminRepo: adminRepo}
}

// What: สร้าง admin ใหม่ — hash password แล้ว save ลง DB
// Why:  admin สร้างได้เฉพาะ internal (ต้องผ่าน adminSecretGuard ก่อนถึงจะเรียก method นี้ได้)
func (a *adminService) RegisterAdmin(ctx context.Context, newAdminReq *dto.CreateAdminRequest, newAdminPassReq string) (*dto.AdminResponse, error) {
	// What: map DTO → domain model
	newAdminDomain := &domain.Admin{
		FirstName: newAdminReq.FirstName,
		LastName:  newAdminReq.LastName,
		Username:  newAdminReq.Username,
		Phone:     newAdminReq.Phone,
		Address:   newAdminReq.Address,
	}
	// What: hash password ก่อน save — ไม่เคย store plain-text
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

// TODO: implement UpdateProfile — อัปเดต profile admin
func (a *adminService) UpdateProfile(ctx context.Context, adminID uint, adminUpdateReq *dto.UpdateAdminRequest) (*dto.AdminResponse, error) {
	return nil, nil
}

// TODO: implement ChangePassword — เปลี่ยน password admin
func (a *adminService) ChangePassword(ctx context.Context, adminID uint, oldPassword, newPassword string) error {
	return nil
}

// TODO: implement GetProfile — ดึงโปรไฟล์ admin
func (a *adminService) GetProfile(ctx context.Context, id uint) (*dto.AdminResponse, error) {
	return nil, nil
}
