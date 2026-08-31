package gormhelper

import (
	"errors"

	"gorm.io/gorm"
)

// IsDuplicateKey บอกว่า error จาก DB เกิดจากการชน unique constraint หรือไม่
//
// WHY มี helper ตัวนี้:
//   - ทุก service เจอปัญหาเดียวกัน — unique violation ถูกโยนเป็น error ธรรมดา
//     แล้วชั้นบนแปลงเป็น 500 Internal Server Error ทั้งที่จริงๆ เป็นความผิดของ input (ควรเป็น 409)
//   - รวมเงื่อนไขไว้ที่เดียว ถ้าวันหน้าเปลี่ยน driver หรือ ORM ก็แก้จุดเดียว
//
// PRECONDITION: ต้องเปิด TranslateError ใน gorm.Config (ดู pkg/utils/database/postgres.go)
// ไม่งั้น GORM จะคืน error ดิบของ driver แล้วฟังก์ชันนี้จะคืน false เสมอ
func IsDuplicateKey(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

// IsForeignKeyViolated บอกว่า error เกิดจากการอ้างถึง record ที่ไม่มีอยู่
// หรือลบ record ที่ยังถูกอ้างถึงอยู่ — ควรแปลงเป็น 409/422 ไม่ใช่ 500
func IsForeignKeyViolated(err error) bool {
	return errors.Is(err, gorm.ErrForeignKeyViolated)
}
