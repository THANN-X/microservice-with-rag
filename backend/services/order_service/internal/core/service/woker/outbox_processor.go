// WHAT: OutboxProcessor เป็น background worker ที่คอย poll outbox table แล้วส่งไป Kafka
//
// HOW: poll ทุก 500ms → ดึง PENDING batch (20 records) → ส่ง Kafka → mark SENT
//      Kafka ล่ม → หยุด batch + พักแบบ backoff (ไม่นับ retry) / อ่าน outbox จาก DB ไม่ได้ → พักแบบเดียวกัน
//      message พังเอง → increment retry_count → dead letter หลัง maxRetries
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
	"errors"
	"fmt"
	"logs"
	"order_service/internal/adapter/messaging/producer"
	repo "order_service/internal/core/port/repo"
	"time"

	"github.com/IBM/sarama"
)

type OutboxProcessor struct {
	repo       repo.OutboxRepository
	producer   producer.EventProducer
	maxRetries int

	// pause ตอน Kafka หรือ DB ใช้ไม่ได้: ข้ามรอบ poll ไปจนถึง pausedUntil แล้วค่อยลองใหม่ (1s → 2s → ... → 30s)
	backoff     time.Duration
	pausedUntil time.Time
}

const (
	pauseBackoffInitial = 1 * time.Second
	pauseBackoffMax     = 30 * time.Second
)

func NewOutboxProcessor(repo repo.OutboxRepository, producer producer.EventProducer) *OutboxProcessor {
	return &OutboxProcessor{
		repo:     repo,
		producer: producer,
		// maxRetries = 5: นับเฉพาะ error ที่เป็นความผิดของ message เอง (ดู isPermanentSendError)
		// Kafka ล่มไม่นับ → ล่มนานแค่ไหน event ก็ไม่ถูก mark FAILED
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
		case now := <-ticker.C:
			if now.Before(p.pausedUntil) {
				continue // Kafka/DB เพิ่งล่ม → ยังไม่ถึงเวลาลองใหม่
			}
			p.afterBatch(now, p.processBatch(ctx))
		}
	}
}

// afterBatch ตั้ง pause ถ้ารอบนี้เจอ Kafka หรือ DB ใช้ไม่ได้ และรีเซ็ตเมื่อกลับมาปกติ
// WHY ต้อง pause? ไม่งั้นทุก 500ms จะลองใหม่ + log error ในขณะที่ Kafka/DB ยังล่มอยู่
// (สาเหตุจริงถูก log ไว้แล้วใน processBatch ตรงนี้บอกแค่ว่าพักนานเท่าไร)
func (p *OutboxProcessor) afterBatch(now time.Time, pause bool) {
	if !pause {
		p.backoff = 0
		return
	}
	p.backoff = min(max(p.backoff*2, pauseBackoffInitial), pauseBackoffMax)
	p.pausedUntil = now.Add(p.backoff)
	logs.Warn(fmt.Sprintf("order outbox: pausing for %s", p.backoff))
}

// processBatch ส่ง PENDING events ทีละตัวตามลำดับ created_at
// คืน true ถ้าควรพักก่อนรอบหน้า: อ่าน outbox จาก DB ไม่ได้ หรือเจอ error ระดับ Kafka (ส่งอะไรก็ไม่ผ่าน)
//
// กติกาที่รักษาลำดับต่อ aggregate (order เดียวกัน):
//   - Kafka ใช้ไม่ได้ → หยุดทั้ง batch ทันที ไม่นับ retry, รอบหน้าเริ่มจาก event แรกเดิม
//   - message พังเอง (เช่นใหญ่เกิน) → นับ retry + block aggregate นั้นในรอบนี้ ส่ง aggregate อื่นต่อได้
//   - aggregate ที่มี event FAILED แล้ว → repository ไม่ดึง event ถัดไปของมันมาอีก (ต้องแก้มือ)
//
// WHY ห้ามข้ามไปส่ง event ถัดไปของ aggregate เดียวกัน?
//   - ถ้า ORDER_CREATED ส่งไม่ผ่านแต่ ORDER_PAID ผ่าน consumer จะได้ PAID ก่อน CREATED
//     → order-history หา order ไม่เจอ → PAID ไปค้างใน DLQ → หน้าเว็บแสดง PENDING ทั้งที่จ่ายแล้ว
func (p *OutboxProcessor) processBatch(ctx context.Context) (pause bool) {
	messages, err := p.repo.GetUnsentMessages(ctx, 20)
	if err != nil {
		logs.Error("Failed to fetch outbox messages: " + err.Error())
		// DB ล่ม → รอบหน้า 500ms ก็พังเหมือนเดิม พักแบบ backoff เหมือนตอน Kafka ล่ม (กัน log ท่วม)
		return true
	}

	// WHY Sequential แทน Concurrent?
	//   - outbox messages เรียง FIFO (created_at ASC) → concurrent อาจทำให้ order เสีย
	//   - Kafka Partition Key (AggregateID) การันตี ordering ใน partition เดียว
	//   - สำหรับ High throughput: แยก Worker Pool ตาม AggregateID แทน concurrent free-for-all
	blocked := map[string]bool{}
	for _, msg := range messages {
		if blocked[msg.AggregateID] {
			continue // event ก่อนหน้าของ aggregate นี้ยังส่งไม่ออก → ตัวนี้ต้องรอ
		}

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
			if !isPermanentSendError(err) {
				// Kafka ใช้ไม่ได้ → message ถัดไปก็จะพังเหมือนกัน หยุดเลยดีกว่า
				// ไม่ IncrementRetryCount เพราะไม่ใช่ความผิดของ message (เดิม Kafka ล่มแค่ ~3 วินาทีก็โดน mark FAILED หมด)
				logs.Error(fmt.Sprintf("Failed to publish message %s, stopping batch: %v", msg.ID, err))
				return true
			}

			// message นี้พังเอง (ส่งกี่รอบก็พังแบบเดิม) → block aggregate นี้ แล้วไปส่ง aggregate อื่นต่อ
			blocked[msg.AggregateID] = true
			logs.Error(fmt.Sprintf("Failed to publish message %s (aggregate %s held this round): %v", msg.ID, msg.AggregateID, err))

			// Dead Letter: ถ้า fail ซ้ำๆ mark FAILED เพื่อ alert + manual intervention
			// หลังจากนี้ event ถัดไปของ aggregate เดียวกันจะไม่ถูกส่งจนกว่าจะแก้ event ที่ FAILED
			if msg.RetryCount >= p.maxRetries {
				if updateErr := p.repo.MarkAsFailed(ctx, msg.ID, err.Error()); updateErr != nil {
					logs.Error("Failed to mark message as failed: " + updateErr.Error())
				}
				logs.Error(fmt.Sprintf("Message %s exceeded max retries, marked as FAILED; later events of aggregate %s are held", msg.ID, msg.AggregateID))
			} else {
				if updateErr := p.repo.IncrementRetryCount(ctx, msg.ID); updateErr != nil {
					logs.Error("Failed to increment retry count: " + updateErr.Error())
				}
			}
			continue
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
	return false
}

// isPermanentSendError แยก error ที่เป็นความผิดของ message เอง (ส่งซ้ำกี่รอบก็พัง) ออกจาก error ระดับ Kafka
// WHY default เป็นชั่วคราว? ถ้าเดาผิดว่าชั่วคราว แค่รอลองใหม่
//
//	แต่ถ้าเดาผิดว่าถาวร event จะถูก mark FAILED ทั้งที่ไม่ควร
func isPermanentSendError(err error) bool {
	var cfgErr sarama.ConfigurationError // เช่น message ใหญ่เกิน Producer.MaxMessageBytes (เช็คฝั่ง client ก่อนส่ง)
	if errors.As(err, &cfgErr) {
		return true
	}
	for _, kerr := range []sarama.KError{
		sarama.ErrMessageSizeTooLarge, // broker ปฏิเสธเพราะใหญ่เกิน
		sarama.ErrInvalidMessage,      // message เสีย (checksum ไม่ตรง)
		sarama.ErrInvalidRecord,       // broker ปฏิเสธ record นี้
		sarama.ErrInvalidTopic,        // ชื่อ topic ใช้ไม่ได้
	} {
		if errors.Is(err, kerr) {
			return true
		}
	}
	return false
}
