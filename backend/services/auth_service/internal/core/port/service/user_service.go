package port

import (
	"auth_service/internal/core/domain"
	req "auth_service/internal/core/port/service/dto"
	res "auth_service/internal/core/port/service/dto"
	"context"
)

type UserService interface {

	// User Management
	RegisterNewUser(ctx context.Context, newUserReq *req.CreateUserRequest, newUserPassReq string) (*res.UserResponse, error)
	UpdateUserInfo(ctx context.Context, userID uint, userUpdateReq *req.UpdateUserRequest) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error

	// Authentication
	GetUserProfile(ctx context.Context, userID uint) (*res.UserResponse, error)
}
