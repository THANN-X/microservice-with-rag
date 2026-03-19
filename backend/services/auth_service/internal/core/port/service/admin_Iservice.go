package port

import (
	dto "auth_service/internal/core/port/service/dto"
	"context"
)

// What: AdminService คือ interface สำหรับ business logic เกี่ยวกับ admin account
// TODO: implement UpdateAdminInfo, UpdatePassword, GetAdminProfile
type AdminService interface {

	// User Management
	RegisterAdmin(ctx context.Context, newAdminReq *dto.CreateAdminRequest, newAdminPassReq string) (*dto.AdminResponse, error)
	UpdateProfile(ctx context.Context, adminID uint, adminUpdateReq *dto.UpdateAdminRequest) (*dto.AdminResponse, error)
	ChangePassword(ctx context.Context, adminID uint, oldPassword, newPassword string) error

	// Authentication
	GetProfile(ctx context.Context, id uint) (*dto.AdminResponse, error)
}
