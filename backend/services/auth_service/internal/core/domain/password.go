package domain

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// What: Password เป็น Value Object ที่ encapsulate hash string ไว้ภายใน
// Why:  ซ่อน bcrypt implementation ออกจาก caller — ทำให้เปลี่ยน hashing algorithm
//
//	ได้ในอนาคตโดยไม่กระทบโค้ดภายนอก
//	field `hash` เป็น unexported เพื่อป้องกันการแก้ไขตรง ๆ จากภายนอก
type Password struct {
	hash string
}

// What: constructor สำหรับโหลด hash ที่มีอยู่แล้ว (เช่น อ่านจาก DB)
// Why:  ต้องมี factory แยก เพราะ hash field เป็น unexported
func NewPassword(hash string) Password {
	return Password{hash: hash}
}

// What: คืน hash string เพื่อนำไปเก็บใน DB
func (p Password) String() string {
	return p.hash
}

// What: รับ plain-text password แล้ว hash ด้วย bcrypt และเก็บผลไว้ใน p.hash
// Why:  ใช้ bcrypt.DefaultCost (10) เพื่อ balance ระหว่าง security และ performance
// TODO: ให้ cost factor อ่านจาก config เพื่อ tune ได้ใน production
func (p *Password) GeneratePassword(rawPassword string) error {
	// What: ตรวจ minimum length ก่อนส่งให้ bcrypt เพื่อ fail-fast
	if len(rawPassword) < 8 {
		return bcrypt.ErrHashTooShort
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.hash = string(hashed)
	return nil
}

// What: เปรียบเทียบ plain-text กับ hash — คืน ErrIncorrectPassword ถ้าไม่ตรง
// Why:  แปลง bcrypt error เป็น domain error เพื่อให้ caller ไม่ต้อง import bcrypt
//
//	และสามารถ assert ด้วย errors.Is(err, domain.ErrIncorrectPassword) ได้
func (p *Password) ComparePassword(rawPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(rawPassword))
	if err != nil {
		// What: map bcrypt mismatch error → domain error เพื่อ decouple
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrIncorrectPassword
		}
		return err
	}
	return nil
}
