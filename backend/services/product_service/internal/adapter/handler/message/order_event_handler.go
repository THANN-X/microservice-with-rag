// WHAT: package message เป็น Kafka Consumer Handler layer ของ product_service
//
//	รับ message จาก topic "order.events" แล้ว route ไปยัง business logic ที่ถูกต้อง
//
// WHY: แยก Kafka concerns (deserialize, route) ออกจาก core service logic
//
//	ทำให้ core service ไม่รู้จัก Kafka เลย (Hexagonal Architecture / Ports & Adapters)
//
// โครงสร้าง:
//
//	Handle()             → entry point, อ่าน EventType แล้ว dispatch
//	handleOrderCreated() → Reserve stock เมื่อมี order ใหม่
//	handleOrderCancelled() → Release stock เมื่อ order ถูก cancel (Saga rollback)
package message

import (
	"context"
	"encoding/json"
	"events"
	"fmt"
	"logs"
	service "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"

	"github.com/IBM/sarama"
)

// WHAT: MessageHandler interface กำหนด contract ของ Kafka message handler
// WHY: ทำให้ consumer.go ไม่ผูกกับ implementation ตรงๆ → unit test ได้ง่ายขึ้น
type MessageHandler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

type productMessageHandler struct {
	cmdService service.ProductCommandService
}

// WHAT: NewProductMessageHandler สร้าง handler พร้อม inject ProductCommandService
// WHY: Dependency Injection ผ่าน constructor ทำให้ swap service impl ได้ตอน test
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
		logs.Warn(fmt.Sprintf("EventType header missing, falling back to body. topic=%s partition=%d offset=%d payload=%s",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Value)))

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
	case "ORDER_CREATED":
		return h.handleOrderCreated(ctx, msg)
	case "ORDER_CANCELLED": // Compensating Transaction → trigger Saga rollback
		return h.handleOrderCancelled(ctx, msg)
	default:
		// ORDER_CONFIRMED, ORDER_PAID, STOCK_* ไม่เกี่ยวกับ product stock — ข้ามไป
		// WHY ไม่ return error? เพราะ message นี้ไม่ได้เป็นของ service เรา ไม่ควร stall consumer
		logs.Warn(fmt.Sprintf("product: unknown event type: %s, skipping. topic=%s partition=%d offset=%d",
			eventType, msg.Topic, msg.Partition, msg.Offset))
		return nil
	}
}

// ─── Sub-Handlers ────────────────────────────────────────────────────────────

// WHAT: handleOrderCreated จัดการ event เมื่อมี order ถูกสร้างใหม่
// WHY: product_service ต้อง reserve stock ของแต่ละ variant ที่อยู่ใน order
//
//	เพื่อป้องกัน oversell ก่อนที่ payment จะ confirm
//	ถ้า reserve ไม่ได้ (stock ไม่พอ) service จะ publish StockReservedEvent{status: "FAILED"}
//	เพื่อให้ order_service rollback Saga
//
// TODO: เพิ่ม retry logic หาก ReserveStock fail ด้วย transient error (เช่น DB timeout)
func (h *productMessageHandler) handleOrderCreated(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var evt events.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return fmt.Errorf("unmarshal ORDER_CREATED: %w", err)
	}

	// WHY อ่าน EventID จาก Header แทน msg.Key?
	//   - Kafka Key = AggregateID (OrderID) ใช้เพื่อรับประกัน partition ordering เท่านั้น
	//   - ORDER_CREATED + ORDER_CANCELLED มี Key เดียวกัน (OrderID) → ใช้เป็น inbox key ไม่ได้
	//   - EventID ใน Header = outbox event UUID ที่ unique ต่อ event → inbox แยกออกจากกันได้
	//   - Fallback: offset-based ID ถ้าไม่มี header (backward compatible)
	 messageID := getHeaderValue(msg.Headers, "EventID")
	if messageID == "" {
		logs.Warn(fmt.Sprintf("product: ORDER_CREATED has no EventID header, falling back to offset-based messageID. topic=%s partition=%d offset=%d",
			msg.Topic, msg.Partition, msg.Offset))
		messageID = fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)
	}

	// WHAT: Map OrderCreatedEvent.Items → []dto.ReserveStockItem
	// WHY: core service ไม่รู้จัก events package → ต้อง map เพื่อรักษา layer boundary
	reqItems := make([]dto.ReserveStockItem, len(evt.Items))
	for i, item := range evt.Items {
		reqItems[i] = dto.ReserveStockItem{
			VariantID: item.VariantID,
			Qty:       item.Quantity,
		}
	}

	req := &dto.ReserveStockReq{
		MessageID: messageID,
		OrderID:   evt.OrderID,
		Items:     reqItems,
	}

	return h.cmdService.ReserveStock(ctx, req)
}

// WHAT: handleOrderCancelled จัดการ event เมื่อ order ถูก cancel
// WHY: Compensating Transaction ใน Saga pattern
//
//	ต้อง release stock ที่เคย reserve ไว้กลับคืน เพื่อให้ stock ถูกต้อง
//	และ order อื่นสามารถ reserve stock นั้นได้
//
// TODO: Log evt.Reason ไว้เพื่อ audit ว่า cancel เพราะอะไร (timeout / user / payment)
func (h *productMessageHandler) handleOrderCancelled(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var evt events.OrderCancelledEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		return fmt.Errorf("unmarshal ORDER_CANCELLED: %w", err)
	}

	// WHY อ่าน EventID จาก Header แทน msg.Key?
	//   - Kafka Key = AggregateID (OrderID) ใช้เพื่อรับประกัน partition ordering เท่านั้น
	//   - ORDER_CREATED + ORDER_CANCELLED มี Key เดียวกัน → ใช้เป็น inbox key ไม่ได้
	//   - EventID ใน Header = outbox event UUID ที่ unique ต่อ event
	//   - Fallback: offset-based ID ถ้าไม่มี header (backward compatible)
	 messageID := getHeaderValue(msg.Headers, "EventID")
	if messageID == "" {
		logs.Warn(fmt.Sprintf("product: ORDER_CANCELLED has no EventID header, falling back to offset-based messageID. topic=%s partition=%d offset=%d",
			msg.Topic, msg.Partition, msg.Offset))
		messageID = fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)
	}

	// WHAT: Map OrderCancelledEvent.Items → []dto.ReserveStockItem (ใช้ struct เดิมได้ เพราะ shape เหมือนกัน)
	reqItems := make([]dto.ReserveStockItem, len(evt.Items))
	for i, item := range evt.Items {
		reqItems[i] = dto.ReserveStockItem{
			VariantID: item.VariantID,
			Qty:       item.Quantity,
		}
	}

	req := &dto.ReserveStockReq{
		MessageID: messageID,
		OrderID:   evt.OrderID,
		Items:     reqItems,
	}

	return h.cmdService.ReleaseStock(ctx, req)
}
