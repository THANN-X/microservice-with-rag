// WHAT: Sarama ConsumerGroupHandler ที่ delegate message processing ให้ MessageHandler
// WHY Strategy Pattern?
//   - แยก Kafka "transport" (รับข้อความ) ออกจาก "business logic" (process แต่ละ event)
//   - ConsumerGroupHandler เป็น generic, MessageHandler เป็น specific per service
//   - ง่ายต่อการ swap messaging system ในอนาคต (เปลี่ยน consumer ไม่กระทบ handler)
package consumer

import (
	"logs"
	"order_service/internal/adapter/handler/message"

	"github.com/IBM/sarama"
)

type ConsumerGroupHandler struct {
	handler message.MessageHandler
}

func NewConsumerGroupHandler(handler message.MessageHandler) sarama.ConsumerGroupHandler {
	return &ConsumerGroupHandler{handler: handler}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error { return nil }

// Cleanup is run at the end of a session
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim วนอ่าน messages จาก Kafka partition ที่ได้รับ assign
// WHY ไม่ใช้ goroutine ใน loop?
//   - Sarama สร้าง goroutine ต่อ 1 partition อยู่แล้ว
//   - Goroutine ใน loop เพิ่ม concurrency → ทำลาย message ordering
// IMPORTANT: ต้อง MarkMessage ทุก message เสมอแม้มี error
//   - ถ้าไม่ mark → Kafka รอ → stall consumer → partition ไม่คืบหน้า
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler.Handle(session.Context(), msg); err != nil {
			// WHY log แต่ไม่ return error?
			//   - Return error จะทำให้ session terminate → rebalance → ช้า
			//   - MarkMessage ยังทำอยู่ → Kafka offset เดิน (skip broken message)
			// TODO: ส่ง broken message ไป Dead Letter Queue แทน skip
			logs.Error(err)
		}
		// Mark offset → Kafka รู้ว่า message นี้ถูก processed แล้ว
		session.MarkMessage(msg, "")
	}
	return nil
}
