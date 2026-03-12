package domain

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Password struct {
	hash string
}

// Factory function to create Password domain model
func NewPassword(hash string) Password {
	return Password{hash: hash}
}

// String method to get the hashed password
func (p Password) String() string {
	return p.hash
}

// Method to generate hashed password from raw password
func (p *Password) GeneratePassword(rawPassword string) error {

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

// Method to compare raw password with hashed password
func (p *Password) ComparePassword(rawPassword string) error {

	err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(rawPassword))

	if err != nil {
		// เช็คว่าถ้าเป็น error รหัสผิด ให้ส่ง Domain Error กลับไป
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			/* ต้อง return error ที่เราสร้างไว้ที่ (domain) */
			return ErrIncorrectPassword
		}
		return err
	}

	return nil
}
