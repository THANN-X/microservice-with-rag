package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres establishes a connection to a PostgreSQL database using the provided DSN.
//
// WHY TranslateError: true
//   - ถ้าไม่เปิด GORM จะคืน error ดิบจาก driver (เช่น "SQLSTATE 23505") ซึ่ง
//     Repository เช็คได้แค่ด้วยการ match string — เปราะและผูกกับ driver
//   - เปิดแล้ว GORM จะแปลงเป็น sentinel error มาตรฐาน (gorm.ErrDuplicatedKey,
//     gorm.ErrForeignKeyViolated) ให้ใช้ errors.Is() ได้ตรงๆ
//   - จำเป็นสำหรับการแยก "ข้อมูลซ้ำ" (ควรเป็น 409) ออกจาก "DB พัง" (500)
//     ไม่งั้น unique violation จะกลายเป็น 500 Internal Server Error หมด
func ConnectPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DryRun:         false,
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
