package port

import (
	dto "auth_service/internal/core/port/service/dto"
	"context"
)

// What: UserService คือ interface สำหรับ business logic เกี่ยวกับ user account
// Why:  handler ใช้ผ่าน interface ทำให้ decouple จาก implementation และ mock ใน test ได้
type UserService interface {

	// User Management
	RegisterUser(ctx context.Context, newUserReq *dto.CreateUserRequest, newUserPassReq string) (*dto.UserResponse, error)
	UpdateUserProfile(ctx context.Context, userID uint, userUpdateReq *dto.UpdateUserRequest) (*dto.UserResponse, error)
	ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error

	// Authentication
	GetProfile(ctx context.Context, userID uint) (*dto.UserResponse, error)
}
