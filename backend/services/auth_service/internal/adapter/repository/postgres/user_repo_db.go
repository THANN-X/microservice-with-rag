package repository

import (
	"context"
	"errors"

	"auth_service/internal/adapter/repository/postgres/entity"
	"auth_service/internal/core/domain"
	port "auth_service/internal/core/port/repo"

	"gorm.io/gorm"
)

// Struct Declaration: Holds the database instance
type userRepositoryDB struct {
	db *gorm.DB
}

// Constructor: Creates a new instance of userRepositoryDB
func NewUserRepositoryDB(db *gorm.DB) port.UserRepository {
	return &userRepositoryDB{db: db}
}

// 3. Method Implementations:
// Each method implements the UserRepository interface
// and contains the data access logic for user operations

// Create adds a new user to the database
func (r *userRepositoryDB) CreateUser(ctx context.Context, user *domain.User) error {
	// Create a new user
	userEntity := entity.ToUserEntity(user)

	if err := r.db.WithContext(ctx).Create(userEntity); err != nil {
		return err.Error
	}
	// Map the generated ID back to the domain user
	user.ID = userEntity.ID
	user.CreatedAt = userEntity.CreatedAt
	user.UpdatedAt = userEntity.UpdatedAt

	return nil
}

// Update modifies an existing user's information
func (r *userRepositoryDB) UpdateUser(ctx context.Context, user *domain.User) error {
	// Update user by ID
	userEntity := entity.ToUserEntity(user)
	// Use WithContext to pass the context
	result := r.db.WithContext(ctx).Save(userEntity)
	// Handle errors
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	user.UpdatedAt = userEntity.UpdatedAt

	return nil
}

// Delete removes a user by ID
func (r *userRepositoryDB) DeleteUser(ctx context.Context, id uint) error {
	// Delete user by ID
	userEntity := &entity.UserEntity{}
	// Use WithContext to pass the context
	result := r.db.WithContext(ctx).Delete(userEntity, id)
	// Handle errors
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return result.Error
	}

	return nil
}

// FindByID retrieves a user by ID
func (r *userRepositoryDB) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	// Find user by ID
	userEntity := &entity.UserEntity{}
	// Use WithContext to pass the context
	result := r.db.WithContext(ctx).First(userEntity, id)
	// Handle errors
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return userEntity.ToUserDomain(), nil
}

// FindByEmail finds a user by their email address
func (r *userRepositoryDB) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Find user by email
	userEntity := &entity.UserEntity{}
	// Use WithContext to pass the context
	result := r.db.WithContext(ctx).First(userEntity, "email = ?", email)
	// Handle errors
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, result.Error
	}

	return userEntity.ToUserDomain(), nil
}
