package service

// What: ChangePasswordReq เป็น input DTO สำหรับเปลี่ยนรหัสผ่าน
// Why:  แยก DTO ออกมาเพื่อ validate rule ต่างหาก และแยก endpoint ออกจาก UpdateUserRequest
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required,min=8"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}
