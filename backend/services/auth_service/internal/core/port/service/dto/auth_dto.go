package service

// What: LoginRequest เป็น input DTO สำหรับ user login
// Why:  DeviceInfo เป็น optional — ใช้เก็บใน session เพื่อ audit และ anomaly detection
type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	DeviceInfo string `json:"device_info"`
	// Why: IPAddress รับจาก client แต่ override ด้วย c.IP() ใน handler อยู่แล้ว
	IPAddress  string `json:"ip_address"`
}

// What: LoginAdminRequest เป็น input DTO สำหรับ admin login
// Why:  ใช้ username แทน email เพราะ admin เป็น internal account
type LoginAdminRequest struct {
	Username   string `json:"username" validate:"required"`
	Password   string `json:"password" validate:"required,min=8"`
	DeviceInfo string `json:"device_info"`
	IPAddress  string `json:"ip_address"`
}

// What: LoginResponse เป็น output DTO หลัง login สำเร็จ
// Why:  client ควรเก็บ refresh_token ไว้อย่างปลอดภัย (เช่น secure HttpOnly cookie)
// TODO: พิจารณาส่ง refresh_token ผ่าน Set-Cookie header แทน response body เพื่อป้องกัน XSS
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// What: RefreshRequest เป็น input DTO สำหรับขอ access token ใหม่
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// What: LogoutRequest เป็น input DTO สำหรับ logout
// Why:  ต้องการ refresh_token เพื่อ identify session ที่จะ revoke
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
