package service

import "auth_service/internal/core/domain"

// Request struct for creating a new admin
type CreateAdminRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

// Request struct for updating admin information
type UpdateAdminRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

type AdminResponse struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Role      string `json:"role"`
}

func ToAdminResponse(a *domain.Admin) *AdminResponse {
	return &AdminResponse{
		ID:        a.ID,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		Username:  a.Username,
		Phone:     a.Phone,
		Address:   a.Address,
		Role:      a.Role,
	}
}
