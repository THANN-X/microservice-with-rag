package repository

import (
	"auth_service/internal/core/domain"

	gormhelper "gorm_helper"
)

// toDomainDBError แปลง error ของ GORM เป็น domain sentinel
// เพื่อให้ Service layer แยก "ข้อมูลซ้ำ" (409) ออกจาก "DB พัง" (500) ได้
// โดยไม่ต้องรู้จัก ORM
func toDomainDBError(err error) error {
	if err == nil {
		return nil
	}
	if gormhelper.IsDuplicateKey(err) {
		return domain.ErrDuplicateKey
	}
	return err
}
