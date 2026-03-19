package port

import (
	"auth_service/internal/core/domain"
	"context"
)

// What: AdminRepository คือ interface สำหรับเข้าถึงข้อมูล admin ใน persistence layer
// TODO: เพิ่ม FindByEmail ถ้า admin รองรับ email-based login ในอนาคต
type AdminRepository interface {

	// Modification
	CreateAdmin(ctx context.Context, admin *domain.Admin) error
	UpdateAdmin(ctx context.Context, admin *domain.Admin) error
	DeleteAdmin(ctx context.Context, id uint) error

	// Retrieval
	FindByUsername(ctx context.Context, username string) (*domain.Admin, error)
	// FindByEmail(ctx context.Context, email string) (*domain.Admin, error)
}
