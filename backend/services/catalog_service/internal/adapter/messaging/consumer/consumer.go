package consumer

import (
	"context"
	"logs"

	"github.com/IBM/sarama"
)

// MessageHandler interface ที่ ConsumerGroupHandler ใช้ dispatch message เข้า handler
// ทุก implementation ต้องรับผิดชอบ idempotency เอง
type MessageHandler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
// ทำงานแบบ sequential per partition — MarkMessage ทุกครั้งไม่ว่าจะ error หรือไม่
// WHY: ป้องกัน consumer stuck ถ้า handler คืน error และ log ไว้เพื่อ investigate
type ConsumerGroupHandler struct {
	handler MessageHandler
}

func NewConsumerGroupHandler(handler MessageHandler) sarama.ConsumerGroupHandler {
	return &ConsumerGroupHandler{handler: handler}
}

func (h *ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler.Handle(session.Context(), msg); err != nil {
			logs.Error(err)
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
