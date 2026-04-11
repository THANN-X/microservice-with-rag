package handler

import (
	"context"
	"encoding/json"
	"events"
	"fmt"
	"logs"
	serviceport "order_history_service/internal/core/port/service"

	"github.com/IBM/sarama"
)

// orderEventHandler รับ Kafka messages จาก "order.events" topic
// และ route ไปยัง OrderHistoryCommandService ตาม event type
type orderEventHandler struct {
	cmdService serviceport.OrderHistoryCommandService
}

func NewOrderEventHandler(cmdService serviceport.OrderHistoryCommandService) *orderEventHandler {
	return &orderEventHandler{cmdService: cmdService}
}

func (h *orderEventHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	eventType := extractEventType(msg)
	if eventType == "" {
		logs.Warn("order-history: received message with no event type — skipping")
		return nil
	}

	messageID := fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)

	switch eventType {

	case "ORDER_CREATED", "OrderCreated":
		var evt events.OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal ORDER_CREATED: %w", err)
		}
		return h.cmdService.HandleOrderCreated(ctx, messageID, &evt)

	case "ORDER_CONFIRMED", "OrderConfirmed":
		var evt events.OrderConfirmedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal ORDER_CONFIRMED: %w", err)
		}
		return h.cmdService.HandleOrderConfirmed(ctx, messageID, &evt)

	case "ORDER_CANCELLED", "OrderCancelled":
		var evt events.OrderCancelledEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal ORDER_CANCELLED: %w", err)
		}
		return h.cmdService.HandleOrderCancelled(ctx, messageID, &evt)

	default:
		return nil
	}
}

// extractEventType อ่าน EventType จาก Kafka header ก่อน ถ้าไม่มีใช้ fallback จาก JSON body
func extractEventType(msg *sarama.ConsumerMessage) string {
	for _, h := range msg.Headers {
		if string(h.Key) == "EventType" {
			return string(h.Value)
		}
	}
	var header events.EventTypeHeader
	if err := json.Unmarshal(msg.Value, &header); err == nil && header.EventType != "" {
		return header.EventType
	}
	return ""
}
