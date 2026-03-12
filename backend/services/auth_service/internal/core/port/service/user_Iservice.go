package port

import (
	"auth_service/internal/core/domain"
	dto "auth_service/internal/core/port/service/dto"
	"context"
)

type UserService interface {

	// User Management
	RegisterNewUser(ctx context.Context, newUserReq *dto.CreateUserRequest, newUserPassReq string) (*dto.UserResponse, error)
	UpdateUserInfo(ctx context.Context, userID uint, userUpdateReq *dto.UpdateUserRequest) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error

	// Authentication
	GetUserProfile(ctx context.Context, userID uint) (*dto.UserResponse, error)
}
