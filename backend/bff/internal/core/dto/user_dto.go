package dto

type UserRawdata struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	// Name  string `json:"name"`
}
type UserRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}
type UserProfileResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	// Name  string `json:"name"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type UserUpdateProfileRequest struct {
	Name string `json:"name" binding:"required"`
}
type UserChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
type UserChangePasswordResponse struct {
	Message string `json:"message"`
}
