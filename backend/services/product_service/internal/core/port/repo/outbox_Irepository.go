package port

import (
	"context"
	"product_service/internal/core/domain"
)

type OutboxRepository interface {
	Save(ctx context.Context, event *domain.OutboxEvent) error

	GetUnsentMessages(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)

	MarkAsSent(ctx context.Context, id string) error

	IncrementRetryCount(ctx context.Context, id string) error

	MarkAsFailed(ctx context.Context, id, errMsg string) error

	// Delete(ctx context.Context, id string) error
}
