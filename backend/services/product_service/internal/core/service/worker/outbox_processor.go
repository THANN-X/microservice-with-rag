package worker

import (
	"context"
	"errors"
	"fmt"
	"logs"
	"product_service/internal/adapter/messaging/producer"
	repo "product_service/internal/core/port/repo"
	"time"

	"github.com/IBM/sarama"
)

// OutboxProcessor เป็น background worker ที่คอย poll outbox table แล้วส่งไป Kafka
//
// HOW: poll ทุก 500ms → ดึง PENDING batch → ส่ง Kafka → mark SENT
//   - Kafka ล่ม → หยุด batch + พักแบบ backoff (ไม่นับ retry)
//   - message พังเอง → retry แล้ว dead letter (FAILED) หลัง maxRetries
//
// WHY ใช้ polling แทน Change Data Capture (CDC) เช่น Debezium?
//   - Polling ง่ายกว่า ไม่ต้อง setup Kafka Connect + connector
//   - Trade-off: latency เพิ่มขึ้นนิดหน่อย (สูงสุด 500ms) แต่ dependency-free
//
// TODO: ต้องการ latency ต่ำ → เพิ่ม channel-based trigger เมื่อ Service เขียน outbox ใหม่
type OutboxProcessor struct {
	repo       repo.OutboxRepository
	producer   producer.EventProducer
	maxRetries int

	// pause ตอน Kafka ใช้ไม่ได้: ข้ามรอบ poll ไปจนถึง pausedUntil แล้วค่อยลองใหม่ (1s → 2s → ... → 30s)
	backoff     time.Duration
	pausedUntil time.Time
}

const (
	brokerBackoffInitial = 1 * time.Second
	brokerBackoffMax     = 30 * time.Second
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
		case now := <-ticker.C:
			if now.Before(p.pausedUntil) {
				continue // Kafka เพิ่งล่ม → ยังไม่ถึงเวลาลองใหม่
			}
			p.afterBatch(now, p.processBatch(ctx))
		}
	}
}

// afterBatch ตั้ง pause ถ้ารอบนี้เจอ Kafka ใช้ไม่ได้ และรีเซ็ตเมื่อกลับมาปกติ
// WHY ต้อง pause? ไม่งั้นทุก 500ms จะลองส่งใหม่ + log error ในขณะที่ Kafka ยังล่มอยู่
func (p *OutboxProcessor) afterBatch(now time.Time, brokerDown bool) {
	if !brokerDown {
		p.backoff = 0
		return
	}
	p.backoff = min(max(p.backoff*2, brokerBackoffInitial), brokerBackoffMax)
	p.pausedUntil = now.Add(p.backoff)
	logs.Warn(fmt.Sprintf("product outbox: kafka unavailable, pausing for %s", p.backoff))
}

// processBatch ส่ง PENDING events ทีละตัวตามลำดับ created_at
// คืน true ถ้าเจอ error ระดับ Kafka (ส่งอะไรก็ไม่ผ่าน) เพื่อให้ Start พักก่อนลองใหม่
//
// กติกาที่รักษาลำดับต่อ aggregate (product เดียวกัน / order เดียวกันของ stock events):
//   - Kafka ใช้ไม่ได้ → หยุดทั้ง batch ทันที ไม่นับ retry, รอบหน้าเริ่มจาก event แรกเดิม
//   - message พังเอง (เช่นใหญ่เกิน) → นับ retry + block aggregate นั้นในรอบนี้ ส่ง aggregate อื่นต่อได้
//   - aggregate ที่มี event FAILED แล้ว → repository ไม่ดึง event ถัดไปของมันมาอีก (ต้องแก้มือ)
//
// WHY ห้ามข้ามไปส่ง event ถัดไปของ aggregate เดียวกัน?
//   - เช่น PRODUCT_CREATED ส่งไม่ผ่านแต่ PRODUCT_PRICE_CHANGED ผ่าน → catalog ได้แก้ราคาสินค้าที่ยังไม่มี
func (p *OutboxProcessor) processBatch(ctx context.Context) (brokerDown bool) {
	// ดึงข้อความค้างส่ง (Batch size 20)
	messages, err := p.repo.GetUnsentMessages(ctx, 20)
	if err != nil {
		logs.Error("Failed to fetch outbox messages: " + err.Error())
		return false
	}

	/* วนลูปส่งทีละตัว (Sequential)
	WHY Sequential แทน Concurrent?
	  - Outbox message มีการ order (created_at ASC FIFO) → concurrent อาจทำให้ order เสีย
	  - Kafka Partition Key (AggregateID) การันตี ordering ใน partition เดียว
	  - TODO: ถ้าต้องการ throughput สูง ให้แยก Worker Pool ตาม AggregateID (same ID = same goroutine) */

	blocked := map[string]bool{}
	for _, msg := range messages {
		if blocked[msg.AggregateID] {
			continue // event ก่อนหน้าของ aggregate นี้ยังส่งไม่ออก → ตัวนี้ต้องรอ
		}

		// เตรียม Header สำหรับ Kafka message
		// WHY ใส่ EventType ใน Header แทนจะรวมใน Body?
		//   - Consumer อ่าน Header ก่อนโดยไม่ต้อง deserialize Body ทั้งก้อน (ประหยัด CPU)
		//   - ทำให้ Consumer routing ทำได้เร็วขึ้น เช่น filter event type ก่อน unmarshal
		headers := map[string]string{
			"EventType": msg.EventType, // สำคัญ! ใส่ type ไว้ให้ Consumer อีกฝั่งแกะ
			"Source":    "product-service",
			// EventID = outbox event UUID ใช้เป็น Inbox idempotency key ใน consumer
			// Kafka key = AggregateID ยังคงเดิม (รับประกัน partition ordering)
			"EventID": msg.ID,
		}

		// Kafka key = AggregateID (รับประกัน partition ordering)
		err := p.producer.Send(msg.Topic, msg.AggregateID, []byte(msg.Payload), headers)

		if err != nil {
			if !isPermanentSendError(err) {
				/* Kafka ใช้ไม่ได้ → message ถัดไปก็จะพังเหมือนกัน หยุดเลยดีกว่า
				ไม่ IncrementRetryCount เพราะไม่ใช่ความผิดของ message
				(เดิม Kafka ล่มแค่ ~3 วินาที message ที่ค้างอยู่ก็โดน mark FAILED หมด) */
				logs.Error(fmt.Sprintf("Failed to publish message %s, stopping batch: %v", msg.ID, err))
				return true
			}

			/* message นี้พังเอง (ส่งกี่รอบก็พังแบบเดิม) → block aggregate นี้ แล้วไปส่ง aggregate อื่นต่อ
			เพิ่ม RetryCount แล้วเช็คว่าเกิน maxRetries หรือยัง:
			  - ไม่เกิน → IncrementRetryCount (รอรอบหน้า)
			  - เกิน   → MarkAsFailed (Dead Letter - ต้อง alert + manual fix)
			            event ถัดไปของ aggregate เดียวกันจะถูกกักไว้จนกว่าจะแก้ */
			blocked[msg.AggregateID] = true
			logs.Error(fmt.Sprintf("Failed to publish message %s (aggregate %s held this round): %v", msg.ID, msg.AggregateID, err))

			if msg.RetryCount >= p.maxRetries {
				p.repo.MarkAsFailed(ctx, msg.ID, err.Error())
				logs.Error(fmt.Sprintf("Message %s moved to DEAD LETTER (Max retries reached); later events of aggregate %s are held", msg.ID, msg.AggregateID))
			} else {
				p.repo.IncrementRetryCount(ctx, msg.ID)
			}
			continue
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
