package service

import "auth_service/internal/core/domain"

// What: CreateAdminRequest เป็น input DTO สำหรับสร้าง admin ใหม่
// TODO: เพิ่ม validate tag เช่น required, min= ในทุก field
type CreateAdminRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

// What: UpdateAdminRequest เป็น input DTO สำหรับอัปเดตข้อมูล admin
// TODO: ยังไม่มี handler/service ใช้ DTO นี้ — implement ต่อไป
type UpdateAdminRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

// What: AdminResponse เป็น output DTO ที่ตัด sensitive field ออก
type AdminResponse struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Role      string `json:"role"`
}

// What: แปลง domain.Admin → AdminResponse ตัด sensitive data ออก
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
