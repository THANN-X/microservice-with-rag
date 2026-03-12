package message

import (
	"context"
	"encoding/json"
	"events"
	"fmt"
	"logs" // package log ของคุณ
	service "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"

	"github.com/IBM/sarama"
)

type MessageHandler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

type productMessageHandler struct {
	cmdService service.ProductCommandService
}

func NewProductMessageHandler(cmdService service.ProductCommandService) MessageHandler {
	return &productMessageHandler{
		cmdService: cmdService,
	}
}

// getHeaderValue ค้นหาค่าของ Kafka Header ตาม key
// WHY ใช้ helper แทนที่จะวน loop ตรงใน Handle?
//   - Handle() จะสะอาดขึ้น และ reuse ได้ถ้าต้องการดึง header หลาย key ในอนาคต
func getHeaderValue(headers []*sarama.RecordHeader, key string) string {
	for _, h := range headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (h *productMessageHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// STEP 1: อ่าน EventType จาก Kafka Header (primary) → fallback ไปแกะ Body
	// WHY check Header ก่อน?
	//   - Header อ่านได้ว่า deserialization Body ทั้งก้อน (ประหยัด CPU)
	//   - ตกลงกับ OutboxProcessor เลยว่า EventType อยู่ใน Header
	eventType := getHeaderValue(msg.Headers, "EventType")

	// STEP 2: Fallback → แกะ Body เพื่อเช็ค EventType ใน payload
	// WHY ต้องมี fallback?
	//   - Producer บาง service อาจเป็น legacy (ไม่ใส่ Header) หรือมี bug ทำให้ลืมใส่
	//   - Fallback ตรวจ Body แทน reject ทันที เพิ่ม resilience
	if eventType == "" {
		var header events.EventTypeHeader

		// Unmarshal เฉพาะส่วนที่ตรงกับ struct EventTypeHeader (Go จะข้าม field อื่นๆ ให้อัตโนมัติ)
		if err := json.Unmarshal(msg.Value, &header); err != nil {
			// ถ้าแกะไม่ออก แสดงว่าเป็น JSON ขยะ หรือ format ผิด
			logs.Error("Failed to extract event type from body: " + err.Error())
			return nil // ข้าม message นี้ไปเลย
		}

		eventType = header.EventType
	}

	if eventType == "" {
		logs.Warn("Event type not found in message")
		return nil
	}

	logs.Info(fmt.Sprintf("Received event: %s topic: %s", eventType, msg.Topic))

	switch eventType {
	case "OrderCreated":
		return h.handleOrderCreated(ctx, msg)
	case "OrderCancelled": // หรือ PaymentFailed → trigger Saga rollback
		return h.handleOrderCancelled(ctx, msg)
	default:
		logs.Warn("Unknown event type: " + eventType)
		// return nil เพื่อให้ Kafka mark offset ว่าอ่านแล้ว
		// WHY ไม่ return error? เพราะ message นี้ไม่ได้เป็นของ service เรา ไม่ควร stall consumer
		return nil
	}
}

// Sub-Handlers
func (h *productMessageHandler) handleOrderCreated(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var evt events.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return err
	}

	// Mapping Event -> DTO ของ Service
	reqItems := make([]dto.ReserveStockItem, len(evt.Items))
	for i, item := range evt.Items {
		reqItems[i] = dto.ReserveStockItem{
			VariantID: item.ProductID, // สมมติ mapping ตรงกัน
			Qty:       item.Quantity,
		}
	}

	req := &dto.ReserveStockReq{
		MessageID: string(msg.Key), // ใช้ Key เป็น MessageID เพื่อทำ Idempotency
		OrderID:   evt.OrderID,
		Items:     reqItems,
	}

	return h.cmdService.ReserveStock(ctx, req)
}

func (h *productMessageHandler) handleOrderCancelled(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// WHAT: Compensating Transaction สำหรับ Saga Rollback
	// WHY ใช้ OrderCreatedEvent struct ซ้ำ?
	//   - สมมติ Order Cancel ส่งข้อมูลเดียวกัน (items list) เพียงแค่ EventType ต่างกัน
	// TODO: สร้าง OrderCancelledEvent struct แยกถ้า payload ไม่เหมือน OrderCreated (e.g. มี cancelReason)
	var evt events.OrderCreatedEvent // สมมติใช้ structure เดิม หรือสร้าง structure ใหม่สำหรับ Cancel
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return err
	}

	reqItems := make([]dto.ReserveStockItem, len(evt.Items))
	for i, item := range evt.Items {
		reqItems[i] = dto.ReserveStockItem{
			VariantID: item.ProductID,
			Qty:       item.Quantity,
		}
	}

	// Logic Release Stock
	req := &dto.ReserveStockReq{
		MessageID: string(msg.Key),
		OrderID:   evt.OrderID,
		Items:     reqItems,
	}

	return h.cmdService.ReleaseStock(ctx, req)
}
