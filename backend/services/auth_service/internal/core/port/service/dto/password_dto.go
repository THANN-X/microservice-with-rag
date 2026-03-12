package service

// Request Structs for User-Related Operations
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required,min=8"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}
