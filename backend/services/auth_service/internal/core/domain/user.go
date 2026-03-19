package domain

import (
	"time"
)

// What: User คือ Domain Model หลักของ auth service แทน "ลูกค้า" ในระบบ
// Why:  เป็น pure struct ไม่มี ORM tag ทำให้ domain layer ไม่ผูกกับ library ใด ๆ
//       (Hexagonal Architecture — domain คือแกนกลาง)
type User struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	// Why: ใช้ pointer เพราะ soft-delete อาจเป็น nil (ยังไม่ถูกลบ)
	DeletedAt *time.Time
	FirstName string
	LastName  string
	Email     string
	// Why: Password ถูก encapsulate เป็น value object เพื่อซ่อน bcrypt logic
	Password  Password
	Phone     string
	Address   string
	Role      string
}

// What: hash password แล้วเก็บไว้ใน Password value object
// Why:  ให้ domain รับผิดชอบ business rule เรื่อง password แทน service layer
func (u *User) SetPassword(rawPassword string) error {
	err := u.Password.GeneratePassword(rawPassword)
	if err != nil {
		return err
	}
	return nil
}

// What: ตรวจสอบว่า raw password ตรงกับ hash ที่เก็บไว้หรือไม่
func (u *User) CheckPassword(rawPassword string) error {
	err := u.Password.ComparePassword(rawPassword)
	if err != nil {
		return err
	}
	return nil
}

// What: เปลี่ยนรหัสผ่าน — ต้องยืนยัน old password ก่อนเสมอ
// Why:  บังคับ verify ไว้ใน domain เพื่อป้องกันการ bypass จาก service layer
func (u *User) ChangePassword(oldPassword, newPassword string) error {
	// What: ตรวจสอบรหัสเก่าก่อน — ถ้าผิดจะ return ErrIncorrectPassword
	err := u.Password.ComparePassword(oldPassword)
	if err != nil {
		return err
	}

	// What: hash รหัสใหม่แล้วอัปเดตลง Password field
	err = u.Password.GeneratePassword(newPassword)
	if err != nil {
		return err
	}
	return nil
}

// What: อัปเดต field ที่แก้ได้ — เฉพาะ field ที่ส่งมาเท่านั้น (partial update)
// Why:  ไม่ใช้ struct assign ตรง ๆ เพื่อป้องกัน field สำคัญ (email, password) ถูกเขียนทับ
// TODO: พิจารณาใช้ functional options หรือ explicit patch struct ถ้า field เพิ่มขึ้นมาก
func (u *User) UpdateUserProfile(req *User) *User {
	if req.FirstName != "" {
		u.FirstName = req.FirstName
	}
	if req.LastName != "" {
		u.LastName = req.LastName
	}
	if req.Address != "" {
		u.Address = req.Address
	}
	if req.Phone != "" {
		u.Phone = req.Phone
	}
	return u
}
