package domain

import "errors"

// What: sentinel errors ของ domain layer
// Why:  ให้ service/handler ใช้ errors.Is() เพื่อ match error type ได้โดยไม่ต้อง
//       compare string — ทำให้ refactor message ภายหลังได้โดยไม่ break logic
// TODO: เพิ่ม ErrEmailAlreadyExists, ErrUsernameAlreadyExists เป็น domain error
//       แทนการ check ซ้ำใน service layer
var (
	// What: user หรือ admin ที่ค้นหาไม่มีใน DB
	ErrUserNotFound = errors.New("user not found")
	// What: plain-text password ไม่ตรงกับ hash ที่เก็บไว้
	ErrIncorrectPassword = errors.New("incorrect old password")
	// What: ไม่พบ session ที่ตรงกับ refresh token ที่ส่งมา
	ErrSessionNotFound = errors.New("session not found")
)
