package command

import (
	"errors"
	"product_service/internal/core/domain"

	"errs"
)

// conflictOrUnexpected แปลง error จาก Repository เป็น HTTP-friendly error
//
// WHY: unique violation คือ input ผิด ไม่ใช่ระบบพัง — ต้องเป็น 409 Conflict
//
//	เดิมทุกอย่างถูกยุบเป็น 500 "Internal Server Error" ทำให้ client
//	ไม่มีทางรู้ว่าซ้ำที่ field ไหน หรือแม้แต่ว่าเป็นเรื่องข้อมูลซ้ำ
//
// conflictMsg ควรบอกให้ชัดว่า field ไหนชน เช่น `SKU "ABC-1" is already used`
func conflictOrUnexpected(err error, conflictMsg string) error {
	if errors.Is(err, domain.ErrDuplicateKey) {
		return errs.NewConflictError(conflictMsg)
	}
	return errs.NewUnexpectedError()
}
