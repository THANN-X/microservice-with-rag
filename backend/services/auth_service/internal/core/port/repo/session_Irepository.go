package port

import (
	"auth_service/internal/core/domain"
	"context"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error)
	RevokeSession(ctx context.Context, refreshToken string) error
}
