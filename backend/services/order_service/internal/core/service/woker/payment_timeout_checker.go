// WHAT: PaymentTimeoutChecker — background worker ที่ auto-cancel order ที่รอชำระนาน เกินกำหนด
//
// WHY ต้องมี timeout checker?
//   - ลูกค้า generate QR Code PromptPay แต่ไม่จ่าย → order ค้างใน AWAITING_PAYMENT ตลอดไป
//   - Stock ถูก reserve อยู่ → สินค้าหมดหน้าร้านโดยไม่จำเป็น
//   - Timeout checker จะ cancel order → trigger OrderCancelledEvent → product_service release stock
//
// WHY ใช้ goroutine แทน cron job?
//   - Simple: ไม่ต้องการ external scheduler (cron) หรือ library
//   - Self-contained: lifecycle ผูกกับ application (ปิด app → worker หยุด ผ่าน ctx.Done())
package worker

import (
	"context"
	"fmt"
	"logs"
	"order_service/internal/core/domain"
	repo "order_service/internal/core/port/repo"
	"time"
)

type PaymentTimeoutChecker struct {
	cmdRepo     repo.OrderCommandRepository
	paymentRepo repo.PaymentRepository
	timeout     time.Duration // เวลาที่รอก่อน cancel (เช่น 30 นาที)
}

func NewPaymentTimeoutChecker(cmdRepo repo.OrderCommandRepository, paymentRepo repo.PaymentRepository, timeout time.Duration) *PaymentTimeoutChecker {
	return &PaymentTimeoutChecker{
		cmdRepo:     cmdRepo,
		paymentRepo: paymentRepo,
		timeout:     timeout,
	}
}

func (c *PaymentTimeoutChecker) Start(ctx context.Context) {
	// WHY ใช้ ticker แทน time.Sleep()?
	//   - ticker ทำงานตรงเวลา (ทุก 1 นาที) ไม่ว่า checkExpired จะใช้เวลานานแค่ไหน
	//   - sleep ทำให้ tick ช้าสะสม (drift) เช่น ถ้า checkExpired ใช้ 10 วินาที → loop ทำงานทุก 70 วิ
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	logs.Info("Payment Timeout Checker started...")

	for {
		select {
		case <-ctx.Done():
			// WHY รอ ctx.Done()?
			//   - เมื่อ main process shutdown (SIGTERM) → ctx ถูก cancel → worker หยุด gracefully
			logs.Info("Payment Timeout Checker stopping...")
			return
		case <-ticker.C:
			c.checkExpired(ctx)
		}
	}
}

func (c *PaymentTimeoutChecker) checkExpired(ctx context.Context) {
	orders, err := c.cmdRepo.FindExpiredPaymentOrders(ctx, c.timeout)
	if err != nil {
		logs.Error("Failed to fetch expired payment orders: " + err.Error())
		return
	}

	for _, order := range orders {
		if err := c.cancelExpiredOrder(ctx, order); err != nil {
			logs.Error(fmt.Sprintf("Failed to cancel expired order %s: %v", order.ID, err))
		}
	}
}

func (c *PaymentTimeoutChecker) cancelExpiredOrder(ctx context.Context, order *domain.Order) error {
	return c.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// WHY Reload order inside TX แทนใช้ order ที่ query มาข้างนอก?
		//   - ระหว่างที่ fetch expired orders กับเข้า TX อาจมี goroutine อื่น update status ไปแล้ว
		//   - ถ้าใช้ order เดิม → อาจ cancel order ที่จ่ายเงินไปแล้วได้ (race condition)
		//   - Reload inside TX + lock row → ได้ข้อมูลล่าสุดเสมอ
		order, err := c.cmdRepo.GetOrderByID(txCtx, order.ID)
		if err != nil {
			return err
		}

		// WHY เช็ค status อีกรอบหลัง reload?
		//   - idempotent guard: ถ้า order ถูก cancel / paid ไปแล้วจากทาง webhook → skip
		if order.Status != domain.OrderStatusAwaitingPayment {
			return nil
		}

		if err := order.MarkPaymentFailed("payment timeout"); err != nil {
			return err
		}

		// WHY ต้อง mark payment record ด้วย?
		//   - order status เปลี่ยนไป CANCELLED แต่ถ้า payment record ยัง PENDING อยู่
		//   - reconciliation report จะเห็น payment ค้าง → mixed signal ทำ accounting ยาก
		//   - ต้อง update payment record ให้ consistent กับ order status
		payment, err := c.paymentRepo.FindByOrderID(txCtx, order.ID)
		// WHY ใช้ err == nil && payment != nil?
		//   - payment record อาจไม่มีถ้า gateway call ล้มเหลวก่อนสร้าง record → ข้ามได้ ไม่ใช่ error
		if err == nil && payment != nil && payment.Status == domain.PaymentStatusPending {
			payment.MarkFailed("payment timeout")
			if err := c.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
				return err
			}
		}

		if err := c.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
			return err
		}

		// SaveDomainEvents: OrderCancelledEvent → product_service จะ release stock
		if err := c.cmdRepo.SaveDomainEvents(txCtx, order); err != nil {
			return err
		}

		logs.Info(fmt.Sprintf("Cancelled expired order %s due to payment timeout", order.ID))
		return nil
	})
}
