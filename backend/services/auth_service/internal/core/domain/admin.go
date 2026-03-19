package domain

import "time"

// What: Admin คือ Domain Model ของผู้ดูแลระบบ
// Why:  แยก Admin ออกจาก User เพราะ business rule ต่างกัน
//       (Admin login ด้วย username, User login ด้วย email)
// TODO: ถ้า business ซับซ้อนขึ้น อาจแยก Admin ออกเป็น service ของตัวเองได้
type Admin struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	FirstName string
	LastName  string
	// Why: ใช้ Username แทน Email เพราะ Admin เป็น internal account ไม่ต้องการ email จริง
	Username  string
	Password  Password
	Phone     string
	Address   string
	Role      string
}

// What: hash password แล้วเก็บไว้ใน Password value object
func (a *Admin) SetPassword(rawPassword string) error {
	err := a.Password.GeneratePassword(rawPassword)
	if err != nil {
		return err
	}
	return nil
}

// What: ตรวจสอบว่า password ที่รับมาตรงกับ hash หรือไม่
func (a *Admin) CheckPassword(rawPassword string) error {
	err := a.Password.ComparePassword(rawPassword)
	if err != nil {
		return err
	}
	return nil
}
