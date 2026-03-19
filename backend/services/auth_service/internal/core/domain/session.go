package domain

import "time"

// What: Session เก็บข้อมูลการ login ของ user/admin แต่ละครั้ง
// Why:  ใช้ server-side session เพื่อให้ revoke refresh token ได้ทันที
//       (stateless JWT อย่างเดียวไม่สามารถ revoke ได้ก่อน token หมดอายุ)
// TODO: เพิ่ม index บน RefreshToken และ ExpiredAt เพื่อ query performance
//       และเพิ่ม cron job หรือ soft-delete cleanup สำหรับ session ที่หมดอายุแล้ว
type Session struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	// Why: UserID และ AdminID เป็น pointer nullable — session เป็นของ user หรือ admin อย่างใดอย่างหนึ่ง
	UserID  *uint
	User    *User
	AdminID *uint
	Admin   *Admin
	// What: token ที่ใช้ขอ access token ใหม่ — unique ต่อ session
	RefreshToken string
	// What: เก็บข้อมูล device ที่ login (เช่น browser, OS) เผื่อใช้ตรวจสอบ suspicious login
	DeviceInfo string
	// What: IP ที่ login มา เผื่อใช้ audit log
	IPAddress string
	// What: เวลาที่ session/token หมดอายุ — ต้องตรงกับ JWT expiry ของ refresh token
	ExpiredAt time.Time
	// What: ถ้าเป็น true หมายความว่า session นี้ถูก logout หรือ revoke แล้ว
	IsRevoked bool
}
