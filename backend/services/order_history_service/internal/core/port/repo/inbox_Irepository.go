package repo

import (
	"context"
	"order_history_service/internal/core/domain"
)

type InboxRepository interface {
	HasProcessed(ctx context.Context, messageID, consumerID string) (bool, error)
	MarkProcessed(ctx context.Context, msg *domain.InboxMessage) error
}
