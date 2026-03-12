package port

import (
	"auth_service/internal/core/domain"
	"context"
)

type AdminRepository interface {

	// Modification
	CreateAdmin(ctx context.Context, admin *domain.Admin) error
	UpdateAdmin(ctx context.Context, admin *domain.Admin) error
	DeleteAdmin(ctx context.Context, id uint) error

	// Retrieval
	FindByUserName(ctx context.Context, username string) (*domain.Admin, error)
	// FindByEmail(ctx context.Context, email string) (*domain.Admin, error)
}
