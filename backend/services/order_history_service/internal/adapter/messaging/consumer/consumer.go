package consumer

import (
	"context"
	"logs"

	"github.com/IBM/sarama"
)

// MessageHandler interface ที่ ConsumerGroupHandler ใช้ dispatch message
type MessageHandler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
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
