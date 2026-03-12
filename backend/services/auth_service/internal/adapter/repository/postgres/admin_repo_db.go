package repository

import (
	"auth_service/internal/adapter/repository/postgres/entity"
	"auth_service/internal/core/domain"
	port "auth_service/internal/core/port/repo"
	"context"
	"errors"

	"gorm.io/gorm"
)

type adminRepositoryDB struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) port.AdminRepository {
	return &adminRepositoryDB{db: db}
}

func (r *adminRepositoryDB) CreateAdmin(ctx context.Context, admin *domain.Admin) error {
	adminEntity := entity.ToAdminEntity(admin)

	if err := r.db.WithContext(ctx).Create(adminEntity); err != nil {
		return err.Error
	}

	admin.ID = adminEntity.ID
	admin.CreatedAt = adminEntity.CreatedAt
	admin.UpdatedAt = adminEntity.UpdatedAt

	return nil
}

func (r *adminRepositoryDB) UpdateAdmin(ctx context.Context, admin *domain.Admin) error {
	adminEntity := entity.ToAdminEntity(admin)

	result := r.db.WithContext(ctx).Save(adminEntity)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	admin.UpdatedAt = adminEntity.UpdatedAt

	return nil
}

func (r *adminRepositoryDB) DeleteAdmin(ctx context.Context, id uint) error {

	adminEntity := &entity.AdminEntity{}

	result := r.db.WithContext(ctx).Delete(adminEntity, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	return nil
}

func (r *adminRepositoryDB) FindByUserName(ctx context.Context, username string) (*domain.Admin, error) {

	adminEntity := &entity.AdminEntity{}

	result := r.db.WithContext(ctx).First(adminEntity, "username = ?", username)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return adminEntity.ToAdminDomain(), nil
}
