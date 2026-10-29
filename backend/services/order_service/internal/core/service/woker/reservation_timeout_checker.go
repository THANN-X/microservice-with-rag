// WHAT: ReservationTimeoutChecker — background worker ที่ยกเลิก order ที่ค้าง PENDING นานเกินกำหนด
//
// WHY ต้องมี?
//   - order ออกจาก PENDING ได้ทางเดียว: product_service จองสต็อกแล้วตอบผลกลับมา (HandleStockResult)
//   - ถ้า ORDER_CREATED ไปจบที่ order.events.dlq (เช่น DB ฝั่ง product ล่มนานเกิน retry)
//     จะไม่มีคำตอบกลับมาเลย → order ค้าง "รอดำเนินการ" ถาวร ลูกค้ากดลองใหม่กี่รอบก็ไม่ผ่าน
//   - PaymentTimeoutChecker ดูแลแค่ AWAITING_PAYMENT ไม่ครอบคลุมเคสนี้
//
// WHY ยกเลิกได้โดยไม่ต้องคืน stock?
//   - ณ เวลานั้น order service ไม่เคยได้ยินว่าจองสำเร็จ → ใช้ ExpireReservation (ไม่ raise OrderCancelledEvent)
//   - ถ้าภายหลังมีคน replay ORDER_CREATED แล้วจองสำเร็จ → HandleStockResult เจอ "SUCCESS + CANCELLED"
//     → RequestStockRelease() คืน stock ให้เองอยู่แล้ว
package worker

import (
	"context"
	"fmt"
	"logs"
	"order_service/internal/core/domain"
	repo "order_service/internal/core/port/repo"
	"time"
)

// reservationTimeoutBatch จำนวน order สูงสุดที่เก็บต่อรอบ ที่เหลือรอรอบถัดไป
const reservationTimeoutBatch = 100

type ReservationTimeoutChecker struct {
	cmdRepo repo.OrderCommandRepository
	timeout time.Duration // order ที่ PENDING นานกว่านี้ถือว่าค้าง
}

func NewReservationTimeoutChecker(cmdRepo repo.OrderCommandRepository, timeout time.Duration) *ReservationTimeoutChecker {
	return &ReservationTimeoutChecker{
		cmdRepo: cmdRepo,
		timeout: timeout,
	}
}

func (c *ReservationTimeoutChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	logs.Info("Reservation Timeout Checker started...")

	for {
		select {
		case <-ctx.Done():
			logs.Info("Reservation Timeout Checker stopping...")
			return
		case <-ticker.C:
			c.checkStale(ctx)
		}
	}
}

func (c *ReservationTimeoutChecker) checkStale(ctx context.Context) {
	orders, err := c.cmdRepo.FindStalePendingOrders(ctx, c.timeout, reservationTimeoutBatch)
	if err != nil {
		logs.Error("Failed to fetch stale pending orders: " + err.Error())
		return
	}

	for _, order := range orders {
		// WHY ไม่หยุดเมื่อพังตัวเดียว? — order แต่ละตัวอยู่คนละ TX ไม่เกี่ยวกัน รอบหน้าจะลองตัวที่พังใหม่เอง
		if err := c.expireOrder(ctx, order.ID); err != nil {
			logs.Error(fmt.Sprintf("Failed to expire pending order %s: %v", order.ID, err))
		}
	}
}

func (c *ReservationTimeoutChecker) expireOrder(ctx context.Context, orderID string) error {
	return c.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// โหลดใหม่ใน TX — ระหว่าง query ข้างนอกกับตอนนี้ order อาจถูก confirm หรือลูกค้ายกเลิกไปแล้ว
		order, err := c.cmdRepo.GetOrderByID(txCtx, orderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderStatusPending {
			return nil
		}

		if err := order.ExpireReservation(); err != nil {
			return err
		}

		// compare-and-set: ถ้า HandleStockResult confirm order ไปแล้วหลังเราโหลด → ไม่เขียนทับ
		// (เขียนทับ CONFIRMED จะทำให้ stock ที่จองไว้จริงค้าง ไม่มีใครคืน)
		updated, err := c.cmdRepo.UpdateOrderStatusIf(txCtx, order.ID, domain.OrderStatusPending, order.Status)
		if err != nil {
			return err
		}
		if !updated {
			logs.Info(fmt.Sprintf("Pending order %s changed status before it could expire, skipping", order.ID))
			return nil
		}

		// SaveDomainEvents: OrderReservationFailedEvent → order_history ปิด order ให้ลูกค้าเห็นว่ายกเลิกแล้ว
		if err := c.cmdRepo.SaveDomainEvents(txCtx, order); err != nil {
			return err
		}

		logs.Warn(fmt.Sprintf("Cancelled order %s: no stock reservation result after %s", order.ID, c.timeout))
		return nil
	})
}
