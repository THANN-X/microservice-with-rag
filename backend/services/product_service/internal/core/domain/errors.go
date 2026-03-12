package domain

import "errors"

// Sentinel errors สำหรับ Domain Layer
//
// WHY ใช้ sentinel errors แทน string:
//   - รองรับ errors.Is() chain ทำให้ Service layer เช็คได้ถูกต้องแม้ error ถูก wrap ด้วย fmt.Errorf("%w", err)
//   - Service layer แปลง domain error เหล่านี้เป็น HTTP-friendly error ผ่าน errs package อีกชั้นหนึ่ง
//     ทำให้ HTTP status code และ message ไม่รั่วออกมาจาก Domain layer
var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNoDataModified   = errors.New("no data was modified") // ใช้แทน RowsAffected = 0 → หมายความว่า record มีอยู่แต่ไม่มีอะไรเปลี่ยน
	ErrInternal         = errors.New("internal server error")
	ErrEmptyProductName = errors.New("product name cannot be empty")
)
