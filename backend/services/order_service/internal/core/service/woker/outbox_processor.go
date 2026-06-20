// WHAT: OutboxProcessor เป็น background worker ที่คอย poll outbox table แล้วส่งไป Kafka
//
// HOW: poll ทุก 500ms → ดึง PENDING batch (20 records) → ส่ง Kafka → mark SENT
//      ถ้าล้ม → increment retry_count → dead letter หลัง maxRetries
//
// WHY ใช้ polling แทน CDC (Change Data Capture เช่น Debezium)?
//   - Polling ง่ายกว่า ไม่ต้อง setup Kafka Connect + connector
//   - ทำงานได้ทันทีโดยไม่ต้องการ infrastructure เพิ่ม
//   - Trade-off: latency เพิ่มขึ้นสูงสุด 500ms (acceptable สำหรับ most use cases)
//
// TODO: ถ้าต้องการ latency ต่ำ → เพิ่ม channel-based trigger เมื่อ service เขียน outbox ใหม่
// TODO: ถ้าต้องการ throughput สูง → แยก Worker Pool ตาม AggregateType (same aggregate = same goroutine)
package worker

import (
	"context"
	"fmt"
	"logs"
	"order_service/internal/adapter/messaging/producer"
	repo "order_service/internal/core/port/repo"
	"time"
)

type OutboxProcessor struct {
	repo       repo.OutboxRepository
	producer   producer.EventProducer
	maxRetries int
}

func NewOutboxProcessor(repo repo.OutboxRepository, producer producer.EventProducer) *OutboxProcessor {
	return &OutboxProcessor{
		repo:     repo,
		producer: producer,
		// maxRetries = 5: หลังจาก retry ครบ → mark FAILED (Dead Letter)
		// TODO: เพิ่ม exponential backoff เพื่อไม่ให้ retry ถี่เกินไปเมื่อ Kafka down
		maxRetries: 5,
	}
}

// Start เริ่ม polling loop
// WHY ต้อง call ใน goroutine (go outboxProcessor.Start(ctx))?
//   - method นี้ block จนกว่า ctx จะ cancel
//   - main.go สั่ง `go outboxProcessor.Start(ctx)` เพื่อให้ HTTP server ทำงานพร้อมกันได้
func (p *OutboxProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	logs.Info("Order Outbox Processor started...")

	for {
		select {
		case <-ctx.Done():
			logs.Info("Order Outbox Processor stopping...")
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *OutboxProcessor) processBatch(ctx context.Context) {
	messages, err := p.repo.GetUnsentMessages(ctx, 20)
	if err != nil {
		logs.Error("Failed to fetch outbox messages: " + err.Error())
		return
	}

	if len(messages) == 0 {
		return // ไม่มีงาน
	}

	// WHY Sequential แทน Concurrent?
	//   - outbox messages เรียง FIFO (created_at ASC) → concurrent อาจทำให้ order เสีย
	//   - Kafka Partition Key (AggregateID) การันตี ordering ใน partition เดียว
	//   - สำหรับ High throughput: แยก Worker Pool ตาม AggregateID แทน concurrent free-for-all
	for _, msg := range messages {
		headers := map[string]string{
			"EventType": msg.EventType, // Consumer อ่าน Header ก่อน deserialize Body (ประหยัด CPU)
			"Source":    "order-service",
			// WHY EventID เป็น outbox event UUID (msg.ID) ไม่ใช่ AggregateID?
			//   - Kafka Key = AggregateID เพื่อรับประกัน ordering (ทุก event ของ order เดียวกันอยู่ partition เดียว)
			//   - EventID ใน header = msg.ID (UUID ที่ unique ต่อ event) ใช้เป็น Inbox idempotency key
			//   - ORDER_CREATED กับ ORDER_CANCELLED มี EventID ต่างกัน จึงไม่ชนกันใน inbox
			"EventID": msg.ID,
		}

		// Kafka key = AggregateID (รับประกัน partition ordering สำหรับทุก event ของ order เดียวกัน)
		err := p.producer.Send(msg.Topic, msg.AggregateID, []byte(msg.Payload), headers)
		if err != nil {
			logs.Error(fmt.Sprintf("Failed to publish message %s: %v", msg.ID, err))

			// Retry logic: increment counter ถ้ายัง < maxRetries
			// WHY ไม่ลบ DB เมื่อส่งไม่ผ่าน?
			//   - ยังต้องการ retry ในรอบถัดไป
			//   - Dead Letter: ถ้า fail ซ้ำๆ mark FAILED เพื่อ alert + manual intervention
			if msg.RetryCount >= p.maxRetries {
				if updateErr := p.repo.MarkAsFailed(ctx, msg.ID, err.Error()); updateErr != nil {
					logs.Error("Failed to mark message as failed: " + updateErr.Error())
				}
				logs.Error(fmt.Sprintf("Message %s exceeded max retries, marked as FAILED", msg.ID))
			} else {
				if updateErr := p.repo.IncrementRetryCount(ctx, msg.ID); updateErr != nil {
					logs.Error("Failed to increment retry count: " + updateErr.Error())
				}
			}
			continue // ข้ามไป message ถัดไป (ไม่ block batch)
		}

		// ส่งสำเร็จ → mark SENT
		if updateErr := p.repo.MarkAsSent(ctx, msg.ID); updateErr != nil {
			// WHY log แต่ไม่ return error?
			//   - Message ถูกส่ง Kafka แล้ว (at-least-once ✓) แต่ DB update ล้ม
			//   - ในรอบถัดไป processor จะ retry → ส่ง Kafka ซ้ำ (duplicate)
			//   - Consumer ฝั่ง order_service มี Inbox Pattern รับมือ duplicate ได้
			logs.Error(fmt.Sprintf("Message %s sent to Kafka but failed to mark as SENT: %v", msg.ID, updateErr))
		}
	}
}
