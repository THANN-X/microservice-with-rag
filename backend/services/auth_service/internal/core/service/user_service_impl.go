package service

import (
	"auth_service/internal/core/domain"
	repo "auth_service/internal/core/port/repo"
	service "auth_service/internal/core/port/service"
	dto "auth_service/internal/core/port/service/dto"
	"context"
	"errors"
	"errs"
	"logs"
)

type userService struct {
	userRepo repo.UserRepository
}

func NewUserService(userRepo repo.UserRepository) service.UserService {
	return &userService{userRepo: userRepo}
}

// Register registers a new user
func (s *userService) RegisterNewUser(ctx context.Context, newUserReq *dto.CreateUserRequest, newUserPassReq string) (*dto.UserResponse, error) {
	// Implementation of user registration logic
	newUserDomain := &domain.User{
		FirstName: newUserReq.FirstName,
		LastName:  newUserReq.LastName,
		Email:     newUserReq.Email,
		Phone:     newUserReq.Phone,
		Address:   newUserReq.Address,
	}

	// Hash the user's password
	if err := newUserDomain.SetPassword(newUserPassReq); err != nil {
		logs.Error(err)
		return nil, errs.NewValidationError("password must be at least 8 characters")
	}

	_, err := s.userRepo.FindByEmail(ctx, newUserDomain.Email)

	if err != nil {
		// ถ้ามี Error และ ไม่ใช่ Not Found คือ DB พัง
		if !errors.Is(err, domain.ErrUserNotFound) {
			logs.Error(err)
			return nil, errs.NewUnexpectedError()
		}

		// ถ้าเป็น Not Found ก็ปล่อยผ่านไป (เพราะเราต้องการให้ Not Found)

	} else {

		// ถ้า err == nil (หาเจอ) -> คือ ซ้ำ
		logs.Error(err)
		return nil, errs.NewConflictError("email already exists")
	}

	// Save user to repository
	if err = s.userRepo.CreateUser(ctx, newUserDomain); err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return dto.ToUserResponse(newUserDomain), nil
}

// UpdateUserProfile updates the user's profile information
func (s *userService) UpdateUserInfo(ctx context.Context, userID uint, userUpdateReq *dto.UpdateUserRequest) (*domain.User, error) {
	// Implementation of user profile update logic

	userDomain := &domain.User{
		FirstName: userUpdateReq.FirstName,
		LastName:  userUpdateReq.LastName,
		Phone:     userUpdateReq.Phone,
		Address:   userUpdateReq.Address,
	}

	existingUser, err := s.userRepo.FindByID(ctx, userID)

	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if existingUser == nil {
		return nil, errs.NewNotFoundError("user not found")
	}

	// Update user fields
	updatedUser := existingUser.UpdateUserProfile(userDomain)

	/* if err != nil {
	// 	logs.Error(err)
	// 	return nil, errs.NewValidationError("invalid user data update")
	// }*/

	// Save updated user to repository
	if err = s.userRepo.UpdateUser(ctx, updatedUser); err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	return updatedUser, nil
}

// ChangePassword changes the user's password
func (s *userService) UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	// Implementation of change password logic
	user, err := s.userRepo.FindByID(ctx, userID)

	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	if user == nil {
		return errs.NewNotFoundError("user not found")
	}

	if err := user.ChangePassword(oldPassword, newPassword); err != nil {
		logs.Error(err)
		if errors.Is(err, domain.ErrIncorrectPassword) {
			return errs.NewUnauthorizedError("incorrect old password")
		}
		return errs.NewValidationError("password change failed")
	}

	// Save updated password to repository
	if err = s.userRepo.UpdateUser(ctx, user); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	return nil
}

// GetUserProfile retrieves the user's profile information
func (s *userService) GetUserProfile(ctx context.Context, id uint) (*dto.UserResponse, error) {
	// Implementation of get user profile logic
	user, err := s.userRepo.FindByID(ctx, id)

	if err != nil {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if user == nil {
		return nil, errs.NewNotFoundError("user not found")
	}

	return dto.ToUserResponse(user), nil
}
