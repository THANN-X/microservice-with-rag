package port

import (
	"bff/internal/core/dto"
	"context"
)

type AIClientPort interface {
	ChatStream(ctx context.Context, message string, sessionID string, ch chan<- *dto.ChatResponseChunk) error
}
