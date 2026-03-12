package service

type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	DeviceInfo string `json:"device_info"`
	IPAddress  string `json:"ip_address"`
}

type LoginAdminRequest struct {
	Username   string `json:"username" validate:"required"`
	Password   string `json:"password" validate:"required,min=8"`
	DeviceInfo string `json:"device_info"`
	IPAddress  string `json:"ip_address"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
