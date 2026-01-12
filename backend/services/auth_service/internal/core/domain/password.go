package domain

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Password struct {
	hash string
}

func NewPassword(hash string) Password {
	return Password{hash: hash}
}

func (p Password) String() string {
	return p.hash
}

func (p *Password) Set(rawPassword string) error {

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

func (p *Password) Check(rawPassword string) error {

	err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(rawPassword))

	if err != nil {
		// เช็คว่าถ้าเป็น error รหัสผิด ให้ส่ง Domain Error ของเรากลับไป
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			// ต้อง return error ที่เราสร้างไว้ในข้อ 1
			// สมมติว่าอยู่ใน package เดียวกัน (domain) ก็เรียกใช้ได้เลย
			return ErrIncorrectPassword
		}
		return err
	}

	return nil
}
