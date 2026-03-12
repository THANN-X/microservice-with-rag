package service

import "auth_service/internal/core/domain"

// Request struct for creating a new user
type CreateUserRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name" validate:"required,min=2"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	Phone     string `json:"phone" validate:"required,min=10"`
	Address   string `json:"address" validate:"required,min=10,max=255"`
}

//	Request struct for updating user information
type UpdateUserRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name" validate:"required,min=2"`
	Phone     string `json:"phone" validate:"required,min=10"`
	Address   string `json:"address" validate:"required,min=10,max=255"`
}

type UserResponse struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Role      string `json:"role"`
}

func ToUserResponse(u *domain.User) *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Phone:     u.Phone,
		Address:   u.Address,
		Role:      u.Role,
	}
}
