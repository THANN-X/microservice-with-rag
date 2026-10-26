package messaging

import (
	"context"
	"fmt"
	"logs"
	"time"

	"github.com/IBM/sarama"
)

// Handler คือ business logic ของแต่ละ service ที่ consumer จะเรียกต่อ message
// ทุก implementation ต้องรับผิดชอบ idempotency เอง (retry + redeliver ทำให้ message เดิมเข้ามาซ้ำได้)
type Handler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
//
// ลำดับเมื่อ handler คืน error:
//  1. error ชั่วคราว → retry ตาม RetryPolicy (exponential backoff)
//  2. Permanent error หรือ retry ครบแล้ว → ส่งเข้า <topic>.dlq แล้ว MarkMessage ไปต่อ
//  3. ส่ง DLQ ไม่สำเร็จ → ไม่ MarkMessage + จบ session → Kafka ส่ง message นี้มาใหม่หลัง rejoin
//
// WHY ข้อ 3 ต้องจบ session แทนการข้ามไป message ถัดไป?
//   - Kafka commit offset แบบ "ถึงตรงนี้แล้ว" ถ้า mark message ถัดไป offset ก็ข้าม message ที่พังไปด้วย
//   - หยุดรอดีกว่าทำ event หาย — DLQ เขียนไม่ได้แปลว่า Kafka เองก็มีปัญหาอยู่แล้ว
//
// WHY ไม่ใช้ goroutine ใน loop?
//   - Sarama สร้าง goroutine ต่อ partition อยู่แล้ว เพิ่มอีกชั้นจะทำ ordering ใน partition เสีย
type ConsumerGroupHandler struct {
	name    string
	handler Handler
	dlq     DeadLetterPublisher
	policy  RetryPolicy
	sleep   func(ctx context.Context, d time.Duration) error
}

// NewConsumerGroupHandler: name ใช้ขึ้นต้น log และบันทึกลง header ของ DLQ ว่า consumer ไหนทำพัง
func NewConsumerGroupHandler(name string, handler Handler, dlq DeadLetterPublisher, policy RetryPolicy) *ConsumerGroupHandler {
	return &ConsumerGroupHandler{
		name:    name,
		handler: handler,
		dlq:     dlq,
		policy:  policy,
		sleep:   sleepCtx,
	}
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := session.Context()
	for msg := range claim.Messages() {
		if err := h.process(ctx, msg); err != nil {
			// rebalance / shutdown ระหว่าง retry → ไม่ mark, เจ้าของ partition คนต่อไปรับไปทำต่อ
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

// process คืน nil เมื่อ message จบแล้ว (สำเร็จ หรือเข้า DLQ แล้ว) ซึ่งแปลว่า MarkMessage ได้
func (h *ConsumerGroupHandler) process(ctx context.Context, msg *sarama.ConsumerMessage) error {
	maxAttempts := h.policy.maxAttempts()

	var err error
	attempts := 0
	for attempts < maxAttempts {
		attempts++
		err = h.handle(ctx, msg)
		if err == nil {
			return nil
		}
		// WHY เช็ค ctx ก่อน? error ที่เกิดเพราะ session ถูก cancel ไม่ใช่ความผิดของ message ห้ามส่งเข้า DLQ
		// NOTE: เช็ค ctx ของ session ไม่ใช่ของ attempt — attempt หมดเวลาถือเป็น error ชั่วคราว ให้ retry ตามปกติ
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if IsPermanent(err) || attempts == maxAttempts {
			break
		}

		wait := h.policy.backoff(attempts)
		logs.Warn(fmt.Sprintf("%s: handle failed, retrying in %s (attempt %d/%d) topic=%s partition=%d offset=%d: %v",
			h.name, wait, attempts, maxAttempts, msg.Topic, msg.Partition, msg.Offset, err))
		if werr := h.sleep(ctx, wait); werr != nil {
			return werr
		}
	}

	logs.Error(fmt.Sprintf("%s: giving up after %d attempt(s), sending to %s topic=%s partition=%d offset=%d: %v",
		h.name, attempts, DeadLetterTopic(msg.Topic), msg.Topic, msg.Partition, msg.Offset, err))

	if dlqErr := h.dlq.Publish(ctx, h.name, msg, err, attempts); dlqErr != nil {
		return fmt.Errorf("%s: publish to %s failed (topic=%s partition=%d offset=%d, original error: %v): %w",
			h.name, DeadLetterTopic(msg.Topic), msg.Topic, msg.Partition, msg.Offset, err, dlqErr)
	}
	return nil
}

// handle เรียก handler 1 ครั้ง ภายใต้ AttemptTimeout ของ policy
func (h *ConsumerGroupHandler) handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	if h.policy.AttemptTimeout <= 0 {
		return h.handler.Handle(ctx, msg)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, h.policy.AttemptTimeout)
	defer cancel()
	return h.handler.Handle(attemptCtx, msg)
}
