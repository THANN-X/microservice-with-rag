package consumer

import (
	"logs"
	"product_service/internal/adapter/handler/message"

	"github.com/IBM/sarama"
)

// ConsumerGroupHandler ใช้ Strategy Pattern → handler inject เข้ามาเพื่อให้ testable
// WHY ไม่เขียน logic ใน ConsumeClaim ตรงๆ?
//   - แยก Kafka "transport" (รับข้อความ) ออกจาก "business logic" (แยก event type และ process)
//   - Kafka consumer layer ที่บางครั้งอาจเปลี่ยนไปใช้ SQS หรือ RabbitMQ โดยไม่ต้องแก้ handler
type ConsumerGroupHandler struct {
	handler message.MessageHandler
}

func NewConsumerGroupHandler(handler message.MessageHandler) sarama.ConsumerGroupHandler {
	return &ConsumerGroupHandler{
		handler: handler,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines exit
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim วนอ่าน messages จาก Kafka partition ที่ได้รับ assign
// WHY ไม่ใช้ goroutine ใน loop?
//   - Sarama สร้าง goroutine ต่อ 1 partition อยู่แล้ว
//   - การเพิ่ม goroutine ในนี้จะเพิ่ม concurrencyแฟนแต่ทำให้ offset ordering เสีย
// IMPORTANT: ต้อง MarkMessage ทุก message เสมอแม้เกิด error มิฉะนั้น Kafka จะ redeliver ทั้ง partition ซัก (stall)
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Note: Do not move the code below to a goroutine.
	for msg := range claim.Messages() {

		// Call the injected handler to process the message
		err := h.handler.Handle(session.Context(), msg)

		if err != nil {
			logs.Error(err)
			// ถ้า Error เราอาจจะไม่ Mark Message ว่าทำเสร็จแล้ว (เพื่อให้ Kafka ส่งมาใหม่ - Retry)
			// หรือถ้าเป็น Error ที่แก้ไม่ได้ (เช่น Json ผิด) อาจจะ Mark ไปเลยเพื่อข้าม
			// ในที่นี้สมมติว่าถ้า Error ให้ข้ามไปก่อน (จริงๆ ควรทำ Retry Topic หรือ Dead Letter Queue)
		}

		// Mark message as processed
		session.MarkMessage(msg, "")
	}
	return nil
}
