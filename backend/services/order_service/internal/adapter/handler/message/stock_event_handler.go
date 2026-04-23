// WHAT: Kafka Message Handler สำหรับ order_service
//       รับ messages จาก topic "stock.events" แล้ว route ไปยัง business logic ที่ถูกต้อง
//
// WHY ไม่เขียน business logic ใน handler โดยตรง?
//   - Handler คือ "adapter" layer → ทำหน้าที่แค่ deserialize + route
//   - Core service ไม่รู้จัก Kafka → ง่ายต่อการ test + swap messaging system
//
// Consumer Topic: "stock.events"
// Events handled:
//   - "STOCK_RESERVED" → HandleStockResult(STATUS: SUCCESS หรือ FAILED)
//   - "STOCK_RELEASED"  → (optional) informational log เท่านั้น
//
// Saga Role: order_service เป็น consumer ของ stock.events (product_service เป็น producer)
package message

import (
	"context"
	"encoding/json"
	"events"
	"fmt"
	"logs"
	service "order_service/internal/core/port/service"
	dto "order_service/internal/core/port/service/dto"

	"github.com/IBM/sarama"
)

// MessageHandler interface สำหรับ consumer.go ใช้ inject handler
// WHY interface?
//   - ConsumerGroupHandler ไม่ผูกกับ concrete message handler
//   - ง่ายต่อการ mock ใน integration test
type MessageHandler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

type orderMessageHandler struct {
	cmdService service.OrderCommandService
}

func NewOrderMessageHandler(cmdService service.OrderCommandService) MessageHandler {
	return &orderMessageHandler{cmdService: cmdService}
}

// getHeaderValue ดึงค่า Kafka Header ตาม key
func getHeaderValue(headers []*sarama.RecordHeader, key string) string {
	for _, h := range headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// Handle คือ entry point ของ Kafka consumer
// WHY อ่าน EventType จาก Header ก่อน?
//   - Header อ่านได้โดยไม่ต้อง deserialize Body ทั้งก้อน (ประหยัด CPU)
//   - Fallback ไปอ่าน Body ถ้า Header ไม่มี (backward compatible กับ legacy producer)
func (h *orderMessageHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// STEP 1: อ่าน EventType จาก Header (primary path)
	eventType := getHeaderValue(msg.Headers, "EventType")

	// STEP 2: Fallback → อ่าน event_type จาก JSON body
	if eventType == "" {
		logs.Warn(fmt.Sprintf("EventType header missing, falling back to body. topic=%s offset=%d", msg.Topic, msg.Offset))

		var header events.EventTypeHeader
		if err := json.Unmarshal(msg.Value, &header); err != nil {
			logs.Error("Failed to extract event type from body: " + err.Error())
			return nil // ข้าม message นี้ (malformed payload → ไม่ควร retry)
		}
		eventType = header.EventType
	}

	if eventType == "" {
		logs.Warn(fmt.Sprintf("Event type not found, skipping. topic=%s offset=%d", msg.Topic, msg.Offset))
		return nil
	}

	logs.Info(fmt.Sprintf("Received event: %s topic: %s offset: %d", eventType, msg.Topic, msg.Offset))

	switch eventType {
	case "STOCK_RESERVED":
		return h.handleStockReserved(ctx, msg)
	case "STOCK_RELEASED":
		// WHY ไม่ทำอะไร?
		//   - STOCK_RELEASED เป็น outcome ของ compensation (order_service trigger release)
		//   - order_service ไม่ต้อง react ต่อ event ของตัวเอง (informational)
		logs.Info(fmt.Sprintf("STOCK_RELEASED received for order, informational only. offset=%d", msg.Offset))
		return nil
	default:
		// WHY ไม่ return error สำหรับ unknown event type?
		//   - Message อาจเป็นของ consumer อื่นที่ subscribe topic เดียวกัน
		//   - Return error จะ stall consumer (ไม่ mark offset) → redeliver ซ้ำๆ ไม่หยุด
		logs.Warn(fmt.Sprintf("Unknown event type: %s, skipping", eventType))
		return nil
	}
}

// handleStockReserved ประมวลผล StockReservedEvent จาก product_service
// รองรับทั้ง Status: "SUCCESS" และ "FAILED" (Saga happy + unhappy path)
func (h *orderMessageHandler) handleStockReserved(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var evt events.StockReservedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return fmt.Errorf("unmarshal StockReservedEvent (topic=%s partition=%d offset=%d): %w",
			msg.Topic, msg.Partition, msg.Offset, err)
	}

	// WHY ใช้ string(msg.Key) เป็น MessageID?
	//   - msg.Key คือ AggregateID ที่ producer (product_service) ตั้งไว้ใน outbox
	//   - สำหรับ StockReservedEvent: AggregateID = fmt.Sprintf("order-%s", orderID)
	//   - ค่านี้ unique ต่อ OrderID → ใช้เป็น InboxEvent.ID เพื่อ Idempotency ได้
	req := &dto.HandleStockResultReq{
		OrderID:   evt.OrderID,
		MessageID: string(msg.Key),
		Status:    evt.Status, // "SUCCESS" หรือ "FAILED"
	}

	return h.cmdService.HandleStockResult(ctx, req)
}
