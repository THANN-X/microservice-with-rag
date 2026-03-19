package port

import (
	"auth_service/internal/core/domain"
	"context"
)

// What: UserRepository คือ interface สำหรับเข้าถึงข้อมูล user ใน persistence layer
// Why:  domain/service ต้อง depend บน interface ไม่ใช่ concrete struct
//       (ทำให้ test ด้วย mock repo ได้โดยไม่ต้องใช้ DB จริง)
// TODO: เพิ่ม FindByEmail หรือ pagination method ถ้า admin ต้องการดูรายชื่อ user ทั้งหมด
type UserRepository interface {

	// Modification
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, user *domain.User) error
	DeleteUser(ctx context.Context, id uint) error

	// Retrieval
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}
