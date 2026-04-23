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
	// STEP 1: อ่าน EventType จาก Kafka Header (primary path)
	eventType := extractEventType(msg)

	// STEP 2: Fallback → อ่าน event_type จาก JSON body พร้อม log บอก location
	if eventType == "" {
		logs.Warn(fmt.Sprintf("order-history: EventType header missing, falling back to body. topic=%s partition=%d offset=%d",
			msg.Topic, msg.Partition, msg.Offset))

		var header events.EventTypeHeader
		if err := json.Unmarshal(msg.Value, &header); err != nil {
			logs.Error(fmt.Sprintf("order-history: failed to extract event type from body: %v payload=%s", err, string(msg.Value)))
			return nil // malformed payload → ไม่ retry
		}
		eventType = header.EventType
	}

	if eventType == "" {
		logs.Warn(fmt.Sprintf("order-history: event type not found, skipping. topic=%s partition=%d offset=%d",
			msg.Topic, msg.Partition, msg.Offset))
		return nil
	}

	logs.Info(fmt.Sprintf("order-history: received event=%s topic=%s partition=%d offset=%d",
		eventType, msg.Topic, msg.Partition, msg.Offset))

	// messageID สร้างจาก topic:partition:offset — unique ต่อ message แน่นอน
	messageID := fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)

	switch eventType {

	case "ORDER_CREATED":
		var evt events.OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal ORDER_CREATED: %w", err)
		}
		return h.cmdService.HandleOrderCreated(ctx, messageID, &evt)

	case "ORDER_CONFIRMED":
		var evt events.OrderConfirmedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal ORDER_CONFIRMED: %w", err)
		}
		return h.cmdService.HandleOrderConfirmed(ctx, messageID, &evt)

	case "ORDER_CANCELLED":
		var evt events.OrderCancelledEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal ORDER_CANCELLED: %w", err)
		}
		return h.cmdService.HandleOrderCancelled(ctx, messageID, &evt)

	case "ORDER_PAID":
		// ORDER_PAID ยังไม่อยู่ใน OrderHistoryCommandService interface
		// TODO: เพิ่ม HandleOrderPaid เมื่อต้องการแสดงสถานะ paid ใน order history
		return nil

	default:
		// STOCK_RESERVED, STOCK_RELEASED, PRODUCT_* ไม่เกี่ยวกับ order history — ข้ามไป
		logs.Warn(fmt.Sprintf("order-history: unknown event type: %s, skipping. topic=%s partition=%d offset=%d",
			eventType, msg.Topic, msg.Partition, msg.Offset))
		return nil
	}
}

// extractEventType อ่าน EventType จาก Kafka Header เท่านั้น
// body fallback จัดการใน Handle() เพื่อให้ log บอก topic/partition/offset ได้
func extractEventType(msg *sarama.ConsumerMessage) string {
	for _, h := range msg.Headers {
		if string(h.Key) == "EventType" {
			return string(h.Value)
		}
	}
	return ""
}
