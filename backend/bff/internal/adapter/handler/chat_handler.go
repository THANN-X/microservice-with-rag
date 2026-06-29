package handler

import (
	"bff/internal/core/dto"
	"bff/internal/core/service"
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type ChatHandler struct {
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) Chat(c *fiber.Ctx) error {
	var req dto.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "message is required",
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Create a context that can be cancelled when the HTTP connection closes
	streamCtx, cancel := context.WithCancel(context.Background())

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer cancel()
		ch := make(chan *dto.ChatResponseChunk)

		go func() {
			defer close(ch)
			_ = h.svc.ChatStream(streamCtx, &req, ch)
		}()

		for chunk := range ch {
			data, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\n", chunk.EventType); err != nil {
				return // client disconnected
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", string(data)); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}
	}))

	return nil
}
