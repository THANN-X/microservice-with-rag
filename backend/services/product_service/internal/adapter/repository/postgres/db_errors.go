package repository

import (
	"product_service/internal/core/domain"

	gormhelper "gorm_helper"
)

// toDomainDBError แปลง error ของ GORM/driver เป็น domain sentinel
//
// WHY: Repository เป็นชั้นเดียวที่ควรรู้จัก error ของ ORM — ชั้นบนเห็นแค่ domain error
//
//	ไม่งั้น unique violation จะไหลขึ้นไปเป็น error ธรรมดาแล้วกลายเป็น 500
//	ทั้งที่จริงๆ เป็นความผิดของ input (ควรเป็น 409 Conflict)
func toDomainDBError(err error) error {
	if err == nil {
		return nil
	}
	if gormhelper.IsDuplicateKey(err) {
		return domain.ErrDuplicateKey
	}
	return err
}
