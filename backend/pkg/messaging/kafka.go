package messaging

import (
	"context"
	"errors"
	"fmt"
	"logs"
	"time"

	"github.com/IBM/sarama"
)

// เพดานการรอระหว่างพยายามเข้ากลุ่มใหม่ตอน Consume() พัง
const (
	consumeRetryInitial = 1 * time.Second
	consumeRetryMax     = 30 * time.Second
)

// KafkaVersion คือเวอร์ชัน protocol ที่ทุก producer/consumer ในระบบใช้คุยกับ broker
// WHY ค่าเดียวทั้งระบบ? ก่อนหน้านี้บาง service ตั้ง V2_8_1_0 บางตัวปล่อย default (V2_1_0_0)
//
//	→ ทำงานได้แต่ feature ที่ใช้ได้ไม่เท่ากัน และไม่มีใครรู้ว่าตัวไหนเป็นตัวไหน
//
// NOTE: ต้องไม่สูงกว่าเวอร์ชันของ broker (cp-kafka 7.5 = Apache Kafka 3.5)
var KafkaVersion = sarama.V2_8_1_0

// LogConsumerErrors อ่าน error จาก consumerGroup.Errors() แล้ว log ออกมา จนกว่า group จะถูก Close
// ต้องตั้ง config.Consumer.Return.Errors = true ไม่งั้น channel นี้จะไม่มี error ส่งเข้ามา
//
// WHY ต้องมีคนอ่าน? sarama ส่ง error เข้า channel แบบไม่รอ ถ้าไม่มีใครอ่าน error จะถูกทิ้งเงียบๆ
//
//	เช่น ConsumeClaim คืน error ตอนเขียน DLQ ไม่ได้ หรือดึง message จาก broker ไม่สำเร็จ
//
// เรียกด้วย `go messaging.LogConsumerErrors(...)` หลังสร้าง consumer group
func LogConsumerErrors(name string, group sarama.ConsumerGroup) {
	for err := range group.Errors() {
		logs.Error(fmt.Sprintf("%s: kafka consumer error: %v", name, err))
	}
}

// RunConsumerGroup วนเรียก group.Consume() ไปเรื่อยๆ จนกว่า ctx จะถูก cancel (shutdown) หรือ group ถูก Close
//   - Consume() return ปกติ (rebalance / session จบ) → วนเข้ากลุ่มใหม่ทันที
//   - Consume() คืน error (เช่น Kafka ล่ม เข้ากลุ่มไม่ได้) → รอ backoff 1s → 2s → ... → 30s แล้วค่อยลองใหม่
//
// WHY ต้องมี backoff? ถ้า Kafka ล่ม Consume() จะพังทันทีทุกรอบ
//
//	ไม่รอเลย = loop หมุนเร็วมาก กิน CPU และ log ท่วม จนหา log อื่นไม่เจอ
//
// WHY ใช้ select ตอนรอ? ต้องรอ 2 อย่างพร้อมกัน — ครบเวลา หรือ shutdown — shutdown ต้องออกได้ทันที ไม่ต้องรอครบ
//
// เรียกด้วย `go messaging.RunConsumerGroup(...)` เพราะ block จน shutdown
func RunConsumerGroup(ctx context.Context, name string, group sarama.ConsumerGroup, topics []string, handler sarama.ConsumerGroupHandler) {
	runConsumerGroup(ctx, name, group, topics, handler, sleepCtx)
}

func runConsumerGroup(ctx context.Context, name string, group sarama.ConsumerGroup, topics []string, handler sarama.ConsumerGroupHandler,
	sleep func(ctx context.Context, d time.Duration) error) {
	wait := consumeRetryInitial
	for {
		err := group.Consume(ctx, topics, handler)
		if ctx.Err() != nil {
			return // shutdown
		}
		if errors.Is(err, sarama.ErrClosedConsumerGroup) {
			return // group ถูก Close ไปแล้ว ลองใหม่กี่รอบก็ไม่มีวันสำเร็จ
		}
		if err == nil {
			wait = consumeRetryInitial // session จบตามปกติ (rebalance) → รีเซ็ต backoff แล้วเข้ากลุ่มใหม่ทันที
			continue
		}

		logs.Error(fmt.Sprintf("%s: kafka consume error, retrying in %s: %v", name, wait, err))
		if sleep(ctx, wait) != nil {
			return // shutdown ระหว่างรอ
		}
		wait = min(wait*2, consumeRetryMax)
	}
}
