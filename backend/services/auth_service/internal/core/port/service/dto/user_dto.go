package service

import "auth_service/internal/core/domain"

// What: CreateUserRequest เป็น input DTO สำหรับสมัคร user ใหม่
// Why:  แยก input struct ออกจาก domain model เพื่อ validate นอกสุดและควบคุมว่าช่องทางไหนบ้างที่รับสิ่งนี้
type CreateUserRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name" validate:"required,min=2"`
	Email     string `json:"email" validate:"required,email"`
	// Why: password อยู่ใน request struct เพื่อรับจาก HTTP body
	//      แต่จะถูกส่งไปเป็น param แยก ไม่เคย embed ไว้ใน domain
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone" validate:"required,min=10"`
	Address  string `json:"address" validate:"required,min=10,max=255"`
}

// What: UpdateUserRequest เป็น input DTO สำหรับอัปเดต profile
// Why:  ไม่รวม email/password เพราะไม่มี endpoint เปลี่ยน email (ควร verify ก่อน)
//       และ password มี endpoint แยกต่างหาก (chgpass)
type UpdateUserRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2"`
	LastName  string `json:"last_name" validate:"required,min=2"`
	Phone    string `json:"phone" validate:"required,min=10"`
	Address  string `json:"address" validate:"required,min=10,max=255"`
}

// What: UserResponse เป็น output DTO ที่ตัด sensitive field ออก
// Why:  ไม่ expose password hash หรือข้อมูล internal อื่น ๆ ไปยัง HTTP response
// TODO: เพิ่ม CreatedAt ถ้า client ต้องการแสดงเวลาได้
type UserResponse struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Role      string `json:"role"`
}

// What: แปลง domain.User → UserResponse ตัด sensitive data ออก
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
