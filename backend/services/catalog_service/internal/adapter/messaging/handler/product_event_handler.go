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
	// STEP 1: อ่าน EventType จาก Kafka Header (primary path)
	eventType := extractEventType(msg)

	// STEP 2: Fallback → อ่าน event_type จาก JSON body พร้อม log บอก location
	if eventType == "" {
		logs.Warn(fmt.Sprintf("catalog: EventType header missing, falling back to body. topic=%s partition=%d offset=%d",
			msg.Topic, msg.Partition, msg.Offset))

		var header events.EventTypeHeader
		if err := json.Unmarshal(msg.Value, &header); err != nil {
			logs.Error(fmt.Sprintf("catalog: failed to extract event type from body: %v payload=%s", err, string(msg.Value)))
			return nil // malformed payload → ไม่ retry
		}
		eventType = header.EventType
	}

	if eventType == "" {
		logs.Warn(fmt.Sprintf("catalog: event type not found, skipping. topic=%s partition=%d offset=%d",
			msg.Topic, msg.Partition, msg.Offset))
		return nil
	}

	logs.Info(fmt.Sprintf("catalog: received event=%s topic=%s partition=%d offset=%d",
		eventType, msg.Topic, msg.Partition, msg.Offset))

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

	case "PRODUCT_IMAGES_UPDATED":
		var evt events.ProductImagesUpdatedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_IMAGES_UPDATED: %w", err)
		}
		return h.cmdService.HandleProductImagesUpdated(ctx, messageID, &evt)

	case "PRODUCT_VARIANT_IMAGES_UPDATED":
		var evt events.ProductVariantImagesUpdatedEvent
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			return fmt.Errorf("unmarshal PRODUCT_VARIANT_IMAGES_UPDATED: %w", err)
		}
		return h.cmdService.HandleProductVariantImagesUpdated(ctx, messageID, &evt)

	default:
		// STOCK_RESERVED, STOCK_RELEASED, ORDER_* ไม่เกี่ยวกับ catalog — ข้ามไป
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
