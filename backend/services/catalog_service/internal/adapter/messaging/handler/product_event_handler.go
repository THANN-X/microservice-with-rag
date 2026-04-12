package handler

import (
	serviceport "catalog_service/internal/core/port/service"
	"context"
	"encoding/json"
	"events"
	"fmt"
	"logs"

	"github.com/IBM/sarama"
)

// productEventHandler รับ Kafka messages จาก "product.events" topic
// และ route ไปยัง CatalogCommandService ตาม event type
type productEventHandler struct {
	cmdService serviceport.CatalogCommandService
}

func NewProductEventHandler(cmdService serviceport.CatalogCommandService) *productEventHandler {
	return &productEventHandler{cmdService: cmdService}
}

func (h *productEventHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	eventType := extractEventType(msg)
	if eventType == "" {
		logs.Warn("catalog: received message with no event type — skipping")
		return nil
	}

	// messageID สร้างจาก topic:partition:offset — unique ต่อ message แน่นอน
	messageID := fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)

	switch eventType {

	case "PRODUCT_CREATED":
		var evt events.ProductCreatedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_CREATED: %w", err)
		}
		return h.cmdService.HandleProductCreated(ctx, messageID, &evt)

	case "PRODUCT_INFO_UPDATED":
		var evt events.ProductInfoUpdatedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_INFO_UPDATED: %w", err)
		}
		return h.cmdService.HandleProductInfoUpdated(ctx, messageID, &evt)

	case "PRODUCT_PRICE_CHANGED":
		var evt events.ProductPriceChangedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_PRICE_CHANGED: %w", err)
		}
		return h.cmdService.HandleProductPriceChanged(ctx, messageID, &evt)

	case "PRODUCT_VARIANT_ADDED":
		var evt events.ProductVariantAddedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_VARIANT_ADDED: %w", err)
		}
		return h.cmdService.HandleProductVariantAdded(ctx, messageID, &evt)

	case "PRODUCT_DELETED":
		var evt events.ProductDeletedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_DELETED: %w", err)
		}
		return h.cmdService.HandleProductDeleted(ctx, messageID, &evt)

	case "STOCK_ADJUSTED":
		var evt events.StockAdjustedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal STOCK_ADJUSTED: %w", err)
		}
		return h.cmdService.HandleStockAdjusted(ctx, messageID, &evt)

	default:
		// STOCK_RESERVED, STOCK_RELEASED, ORDER_* ไม่เกี่ยวกับ catalog — ข้ามไป
		return nil
	}
}

// extractEventType อ่าน EventType จาก Kafka header ก่อน
// ถ้าไม่มี header ใช้ fallback อ่านจาก JSON body
func extractEventType(msg *sarama.ConsumerMessage) string {
	for _, h := range msg.Headers {
		if string(h.Key) == "EventType" {
			return string(h.Value)
		}
	}
	// Fallback: peek event_type from JSON body
	var header events.EventTypeHeader
	if err := json.Unmarshal(msg.Value, &header); err == nil && header.EventType != "" {
		return header.EventType
	}
	return ""
}
