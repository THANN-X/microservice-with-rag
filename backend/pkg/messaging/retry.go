package messaging

import (
	"context"
	"time"
)

// RetryPolicy กำหนดว่า handler จะถูกเรียกซ้ำกี่ครั้ง และรอนานเท่าไหร่ระหว่างรอบ
// WHY retry ในตัว consumer แทน retry topic?
//   - ง่ายกว่ามาก ไม่ต้องสร้าง topic + consumer เพิ่มต่อ service
//   - error ส่วนใหญ่ในระบบนี้คือ DB/network สะดุดสั้นๆ ซึ่งหายภายในไม่กี่วินาที
//
// TRADE-OFF: ระหว่าง retry partition นั้นหยุดรอ (ค่า default: backoff รวม 3.5s + handler ไม่เกิน 5s ต่อครั้ง)
//
//	แลกกับการรักษา ordering ของ event ใน partition ไว้ ซึ่ง Saga ต้องการ
type RetryPolicy struct {
	MaxAttempts    int           // จำนวนครั้งที่เรียก handler ทั้งหมด (รวมครั้งแรก)
	InitialBackoff time.Duration // รอหลังพังครั้งแรก แล้วคูณ 2 ทุกรอบ
	MaxBackoff     time.Duration // เพดานของการรอแต่ละรอบ

	// AttemptTimeout เพดานเวลาของ handler แต่ละครั้ง (0 = ไม่จำกัด)
	// WHY: ถ้าไม่มี handler รอนานเท่าที่ driver ยอมรอ — Mongo รอหา server 30s ต่อครั้ง
	//      → retry 4 ครั้งกลายเป็น partition ค้าง ~2 นาที แทนที่จะเป็นไม่กี่วินาที (เจอตอนทดสอบปิด Mongo จริง)
	// WHY ต่อครั้ง ไม่ใช่ต่อ message? ทุก retry ได้เวลาเต็ม ครั้งแรกที่ช้าไม่กินเวลาของครั้งถัดไป
	// NOTE: handler ต้องส่ง ctx ต่อลงไปถึง DB call ทุกตัว timeout นี้ถึงจะมีผล
	AttemptTimeout time.Duration
}

// DefaultRetryPolicy: 4 ครั้ง รอ 0.5s → 1s → 2s, แต่ละครั้งให้เวลาไม่เกิน 5s
// → message ที่ dependency ล่มจะเข้า DLQ ภายใน ~23.5s (4×5s + 3.5s) แทน ~2 นาที
//
// WHY 5s? งานของ handler คือ query/transaction เล็กๆ ปกติจบในหลักมิลลิวินาที
//
//	5s ถือว่าผิดปกติแล้ว และถ้าโดนตัดกลาง transaction DB จะ rollback + inbox กันการทำซ้ำตอน retry
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    4,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		AttemptTimeout: 5 * time.Second,
	}
}

// backoff คืนเวลารอหลังพังครั้งที่ attempt (เริ่มที่ 1)
func (p RetryPolicy) backoff(attempt int) time.Duration {
	d := p.InitialBackoff
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= p.MaxBackoff {
			return p.MaxBackoff
		}
	}
	return min(d, p.MaxBackoff)
}

func (p RetryPolicy) maxAttempts() int {
	return max(p.MaxAttempts, 1)
}

// sleepCtx รอ d หรือจนกว่า ctx ถูก cancel (rebalance / shutdown) แล้วแต่อย่างไหนมาก่อน
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
