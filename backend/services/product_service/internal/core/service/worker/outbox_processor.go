package worker

import (
	"context"
	"fmt"
	"logs"
	"product_service/internal/adapter/messaging/producer"
	repo "product_service/internal/core/port/repo"
	"time"
)

// OutboxProcessor เป็น background worker ที่คอย poll outbox table แล้วส่งไป Kafka
//
// HOW: poll ทุก 500ms → ดึง PENDING batch → ส่ง Kafka → mark SENT → retry/dead letter ถ้าล้ม
// WHY ใช้ polling แทน Change Data Capture (CDC) เช่น Debezium?
//   - Polling ง่ายกว่า ไม่ต้อง setup Kafka Connect + connector
//   - Trade-off: latency เพิ่มขึ้นนิดหน่อย (สูงสุด 500ms) แต่ dependency-free
//
// TODO: ต้องการ latency ต่ำ → เพิ่ม channel-based trigger เมื่อ Service เขียน outbox ใหม่
type OutboxProcessor struct {
	repo       repo.OutboxRepository
	producer   producer.EventProducer
	maxRetries int
}

func NewOutboxProcessor(repo repo.OutboxRepository, producer producer.EventProducer) *OutboxProcessor {
	return &OutboxProcessor{
		repo:     repo,
		producer: producer,
		// maxRetries = 5: หลังจากนี้ message จะถูก mark FAILED (Dead Letter)
		// TODO: ทำ exponential backoff เพื่อไม่ให้ retry ถี่เกินไป เมื่อ Kafka down
		maxRetries: 5,
	}
}

// Start begins polling for unsent outbox messages.
// WHY ต้องเรียกใน goroutine?
//   - ศัพท์ call นี้ block ตลอด (infinite loop) จนกว่า ctx จะถูก cancel
//   - main.go สั่ง `go outboxProcessor.Start(ctx)` เพื่อให้ HTTP server ยังรับคำขอได้
func (p *OutboxProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	logs.Info("Outbox Processor started...")

	for {
		select {
		case <-ctx.Done():
			logs.Info("Stopping Outbox Processor...")
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *OutboxProcessor) processBatch(ctx context.Context) {
	// ดึงข้อความค้างส่ง (Batch size 20)
	messages, err := p.repo.GetUnsentMessages(ctx, 20)
	if err != nil {
		logs.Error("Failed to fetch outbox messages: " + err.Error())
		return
	}

	if len(messages) == 0 {
		return // ไม่มีงาน
	}

	/* วนลูปส่งทีละตัว (Sequential)
	WHY Sequential แทน Concurrent?
	  - Outbox message มีการ order (created_at ASC FIFO) → concurrent อาจทำให้ order เสีย
	  - Kafka Partition Key (AggregateID) การันตี ordering ใน partition เดียว
	  - TODO: ถ้าต้องการ throughput สูง ให้แยก Worker Pool ตาม AggregateID (same ID = same goroutine) */

	for _, msg := range messages {
		// เตรียม Header สำหรับ Kafka message
		// WHY ใส่ EventType ใน Header แทนจะรวมใน Body?
		//   - Consumer อ่าน Header ก่อนโดยไม่ต้อง deserialize Body ทั้งก้อน (ประหยัด CPU)
		//   - ทำให้ Consumer routing ทำได้เร็วขึ้น เช่น filter event type ก่อน unmarshal

		headers := map[string]string{
			"EventType": msg.EventType, // สำคัญ! ใส่ type ไว้ให้ Consumer อีกฝั่งแกะ
			"Source":    "product-service",
		}

		/*กำหนด Topic
		อาจจะเขียน logic แยก Topic ตรงนี้ หรือจะเก็บ Topic ไว้ใน DB Outbox เลยก็ได้
		สมมติ: ถ้า AggregateType เป็น STOCK ให้ส่ง topic "stock.events" ถ้า PRODUCT ส่ง "product.events"
		topic := "product.events"
		if msg.AggregateType == "STOCK" {
			topic = "stock.events"
		}*/

		// ส่งเข้า Kafka
		err := p.producer.Send(msg.Topic, msg.AggregateID, []byte(msg.Payload), headers)

		if err != nil {
			logs.Error("Failed to publish message: " + err.Error())

			/* ส่งไม่ผ่าน → ไม่ต้องลบ DB ทิ้ง
			รอ OutboxProcessor รอบหน้ามาหยิบใหม่ (Retry at-least-once)
			เพิ่ม RetryCount แล้วเช็คว่าเกิน maxRetries หรือยัง:
			  - ไม่เกิน → IncrementRetryCount (รอรอบหน้า)
			  - เกิน   → MarkAsFailed (Dead Letter Queue - ต้อง alert + manual fix) */

			if msg.RetryCount >= p.maxRetries {
				p.repo.MarkAsFailed(ctx, msg.ID, err.Error())
				logs.Error(fmt.Sprintf("Message %s moved to DEAD LETTER (Max retries reached)", msg.ID))

			} else {
				/* ยังไม่เกิน maxRetries → เพิ่ม RetryCount รอมาหยิบรอบหน้า
				TODO: ทำ Exponential Backoff โดยเลื่อน updated_at หรือเพิ่ม next_retry_at field
				เพื่อไม่ให้ Processor หยิบ message เดิมซ้ำถี่เกินไปตอน Kafka ล่ม */
				p.repo.IncrementRetryCount(ctx, msg.ID)
			}
			continue // ไปส่งข้อความถัดไป
		}

		/* ส่งผ่านแล้ว → mark SENT แทนการลบ
		WHY เก็บ record ไว้แทนที่จะลบทิ้ง?
		  - ใช้เป็น audit trail ว่า event ไหนถูกส่งไปแล้ว เมื่อไร
		  - ถ้าอยากลบทิ้งเพื่อประหยัด disk ให้สร้าง cleanup job ลบ SENT records ที่เก่ากว่า N วัน
		  - TODO: implement cleanup job ดังกล่าว */

		if err := p.repo.MarkAsSent(ctx, msg.ID); err != nil {
			logs.Error("Failed to mark message as sent: " + err.Error())

		} else {
			logs.Info(fmt.Sprintf("Outbox message %s sent successfully", msg.ID))
		}

		/* ส่งผ่านแล้ว ลบออกจาก DB
			if err := p.repo.Delete(ctx, msg.ID); err != nil {
				logs.Error("Failed to delete outbox message: " + err.Error())
			}
		}*/
	}
}
