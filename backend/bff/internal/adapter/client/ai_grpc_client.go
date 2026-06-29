package client

import (
	"bff/internal/adapter/client/aipb"
	"bff/internal/core/dto"
	"bff/internal/core/port"
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AIGRPCClient struct {
	conn   *grpc.ClientConn
	client aipb.AIChatServiceClient
}

func NewAIGRPCClient(address string) (port.AIClientPort, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AI service at %s: %w", address, err)
	}

	return &AIGRPCClient{
		conn:   conn,
		client: aipb.NewAIChatServiceClient(conn),
	}, nil
}

func (c *AIGRPCClient) ChatStream(ctx context.Context, message string, sessionID string, ch chan<- *dto.ChatResponseChunk) error {
	stream, err := c.client.Chat(ctx, &aipb.ChatRequest{
		Message:   message,
		SessionId: sessionID,
	})
	if err != nil {
		ch <- &dto.ChatResponseChunk{
			EventType:   "error",
			TextContent: "AI service connection error: " + err.Error(),
		}
		return fmt.Errorf("AI service chat stream error: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			ch <- &dto.ChatResponseChunk{
				EventType:   "error",
				TextContent: err.Error(),
			}
			return fmt.Errorf("error receiving from AI service stream: %w", err)
		}

		ch <- &dto.ChatResponseChunk{
			EventType:   resp.EventType,
			TextContent: resp.TextContent,
			ProductIDs:  resp.ProductIds,
		}
	}

	return nil
}

func (c *AIGRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
