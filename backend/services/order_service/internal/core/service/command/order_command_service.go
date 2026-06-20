// WHAT: OrderCommandService implementation — ประมวลผล write use cases ทั้งหมดสำหรับ Order
//
// WHY ต้องมี 3 repositories?
//   - cmdRepo    : เขียน/อ่าน Order aggregate (primary data store)
//   - outboxRepo : อ่าน outbox สำหรับ OutboxProcessor (ไม่ได้ใช้ใน command service โดยตรง)
//     แต่ SaveDomainEvents อยู่ใน cmdRepo → outboxRepo ส่งต่อให้ processor
//   - inboxRepo  : ตรวจสอบ/บันทึก processed message IDs (Inbox Pattern / Idempotency)
//
// Saga Flow ที่ service รองรับ:
//
//	PlaceOrder → [OrderCreatedEvent] → product_service reserves stock
//	HandleStockResult(SUCCESS) → ConfirmOrder → [OrderConfirmedEvent]
//	HandleStockResult(FAILED)  → MarkReservationFailed → CANCELLED (no event)
//	HandleStockResult(SUCCESS) + order CANCELLED → RequestStockRelease → [OrderCancelledEvent]
//	CancelOrder (CONFIRMED)    → Cancel → [OrderCancelledEvent] → product_service releases stock
package command

import (
	"context"
	"errors"
	"errs"
	"fmt"
	"logs"
	"order_service/internal/core/domain"
	gateway "order_service/internal/core/port/gateway"
	repo "order_service/internal/core/port/repo"
	service "order_service/internal/core/port/service"
	dto "order_service/internal/core/port/service/dto"
	"order_service/internal/core/port/service/mapper"
)

type orderCommandService struct {
	cmdRepo        repo.OrderCommandRepository
	inboxRepo      repo.InboxRepository
	paymentRepo    repo.PaymentRepository
	paymentGateway gateway.PaymentGateway
	catalogClient  gateway.CatalogClient
}

func NewOrderCommandService(cmdRepo repo.OrderCommandRepository, inboxRepo repo.InboxRepository, paymentRepo repo.PaymentRepository, paymentGateway gateway.PaymentGateway, catalogClient gateway.CatalogClient) service.OrderCommandService {
	return &orderCommandService{
		cmdRepo:        cmdRepo,
		inboxRepo:      inboxRepo,
		paymentRepo:    paymentRepo,
		paymentGateway: paymentGateway,
		catalogClient:  catalogClient,
	}
}

// ─── USE CASE: Place Order ────────────────────────────────────────────────────

// PlaceOrder สร้าง Order ใหม่ และ raise OrderCreatedEvent เพื่อเริ่ม Saga stock reservation
//
// WHY ใช้ RunInTx ครอบทั้ง CreateOrder + SaveDomainEvents?
//   - Transactional Outbox: Order row และ outbox event ต้องอยู่ใน TX เดียวกัน
//   - ถ้า TX fail → ทั้งคู่ rollback → ไม่มี orphan event ที่ reference order ที่ไม่มีอยู่
//   - ถ้า crash หลัง TX commit → OutboxProcessor จะ retry ส่ง event ให้ใหม่
//
// WHY aggregate-first (UUID domain-generated)?
//   - domain.NewOrder() สร้าง UUID ก่อน DB → PlaceOrder() raise event with correct ID ทันที
//   - ต่างจาก pattern ที่ต้อง "persist ก่อน เพื่อรู้ ID" (เช่น AddVariant ใน product_service)
func (s *orderCommandService) PlaceOrder(ctx context.Context, customerID uint, req *dto.CreateOrderReq) (*dto.OrderRes, error) {
	// ดึง price + product snapshot จาก catalog_service (server-side — ป้องกัน price tampering)
	// WHY loop แยกแทน batch?
	//   - catalog_service ยังไม่มี batch endpoint; ค่าเฉลี่ย เพราะ order มี item ไม่กี่ item (ปกติ 1-5)
	//   - TODO: เพิ่ม GET /catalog/variants/batch?ids=1,2,3 ถ้าต้องการ optimize
	items := mapper.ToOrderItemsDomain(req.Items)
	for i, reqItem := range req.Items {
		snap, err := s.catalogClient.GetVariantSnapshot(ctx, reqItem.VariantID)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch variant %d from catalog: %w", reqItem.VariantID, err)
		}
		items[i].UnitPrice = snap.Price
		items[i].ProductName = snap.ProductName
		items[i].VariantName = snap.VariantName
		items[i].ImageURL = snap.ImageURL
	}

	address := mapper.ToShippingAddressDomain(req.ShippingAddress)

	// Factory: validate invariants + สร้าง aggregate พร้อม UUIDs
	order, err := domain.NewOrder(customerID, items, address, req.Note)
	if err != nil {
		// Domain validation error → 400 Bad Request
		return nil, errs.NewValidationError(err.Error())
	}

	// Business action: raise OrderCreatedEvent (triggers Saga)
	// WHY แยก NewOrder จาก PlaceOrder?
	//   - NewOrder = pure factory (safe to call multiple times in test)
	//   - PlaceOrder = one-way action with side effect (raise event) → call once
	order.PlaceOrder()

	// Atomic: save Order + Items + outbox event ใน TX เดียว
	if err := s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.cmdRepo.CreateOrder(txCtx, order); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}
		if err := s.cmdRepo.SaveDomainEvents(txCtx, order); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return mapper.ToOrderRes(order), nil
}

// ─── USE CASE: Cancel Order (Customer) ───────────────────────────────────────

// CancelOrder ยกเลิก Order โดย customer
//
// WHY ต้อง Load aggregate ก่อน?
//  1. Ownership check: ต้องตรวจว่า order.CustomerID == customerID ก่อน (prevent IDOR)
//  2. Domain method Cancel() enforce state transition invariant
//  3. ถ้า CONFIRMED → domain raise OrderCancelledEvent ใน aggregate
//
// WHY ไม่ do simple targeted UPDATE (SET status=CANCELLED WHERE id=? AND customer_id=?)?
//   - ข้ามไป domain → ไม่มี invariant enforcement (e.g. ถ้า COMPLETED ก็จะถูก cancel ได้)
//   - ไม่มี OrderCancelledEvent → product_service ไม่รู้ว่าต้อง release stock
func (s *orderCommandService) CancelOrder(ctx context.Context, orderID string, customerID uint, reason string) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		order, err := s.cmdRepo.GetOrderByID(txCtx, orderID)
		if err != nil {
			if errors.Is(err, domain.ErrOrderNotFound) {
				return errs.NewNotFoundError(fmt.Sprintf("order %s not found", orderID))
			}
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		// Ownership check (domain-level authorization)
		// WHY ทำใน service แทน handler?
		//   - Authorization ที่เกี่ยวกับ aggregate data ควรอยู่ใกล้ data access
		//   - Handler ไม่ควรรู้ว่า Order มี CustomerID (มันแค่ส่ง customerID จาก token)
		if order.CustomerID != customerID {
			return errs.NewForbiddenError("you do not have permission to cancel this order")
		}

		// Refund ถ้า order ถูกจ่ายเงินไปแล้ว
		if err := s.refundIfPaid(txCtx, order); err != nil {
			return err
		}

		if err := order.Cancel(reason); err != nil {
			if errors.Is(err, domain.ErrOrderAlreadyCancelled) {
				return errs.NewConflictError("order is already cancelled")
			}
			if errors.Is(err, domain.ErrCannotCancelCompletedOrder) {
				return errs.NewForbiddenError("cannot cancel a completed order")
			}
			return errs.NewValidationError(err.Error())
		}

		if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		// SaveDomainEvents จะ save OrderCancelledEvent (ถ้ามี) ลง outbox
		// ถ้า PENDING → Cancel() ไม่ raise event → PopDomainEvents คืน [] → no-op
		if err := s.cmdRepo.SaveDomainEvents(txCtx, order); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// ─── USE CASE: Admin Cancel Order ────────────────────────────────────────────

// AdminCancelOrder ยกเลิก Order โดย admin (ไม่มี ownership check)
// WHY แยก AdminCancelOrder จาก CancelOrder?
//   - Admin ไม่มี customerID → ไม่ควรส่ง customerID แปลกๆ เพื่อ bypass ownership
//   - Permission model ชัดเจน: AdminGuard middleware block non-admin ก่อนถึง handler นี้
func (s *orderCommandService) AdminCancelOrder(ctx context.Context, orderID string, reason string) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		order, err := s.cmdRepo.GetOrderByID(txCtx, orderID)
		if err != nil {
			if errors.Is(err, domain.ErrOrderNotFound) {
				return errs.NewNotFoundError(fmt.Sprintf("order %s not found", orderID))
			}
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		// Refund ถ้า order ถูกจ่ายเงินไปแล้ว
		if err := s.refundIfPaid(txCtx, order); err != nil {
			return err
		}

		if err := order.Cancel(reason); err != nil {
			if errors.Is(err, domain.ErrOrderAlreadyCancelled) {
				return errs.NewConflictError("order is already cancelled")
			}
			if errors.Is(err, domain.ErrCannotCancelCompletedOrder) {
				return errs.NewForbiddenError("cannot cancel a completed order")
			}
			return errs.NewValidationError(err.Error())
		}

		if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		if err := s.cmdRepo.SaveDomainEvents(txCtx, order); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// ─── USE CASE: Handle Stock Result (Saga Continuation) ───────────────────────

// HandleStockResult ประมวลผล StockReservedEvent จาก product_service
//
// WHY มี Inbox Pattern?
//   - Kafka at-least-once → StockReservedEvent อาจถูกส่งซ้ำ (consumer rebalance)
//   - ถ้าไม่มี inbox → ConfirmOrder ถูกเรียกซ้ำ → domain คืน ErrInvalidOrderTransition → harmless แต่ noisy
//   - With inbox → skip replay cleanly
//
// State machine สำหรับ req.Status:
//
//	"SUCCESS" + PENDING → ConfirmOrder() → CONFIRMED + OrderConfirmedEvent
//	"SUCCESS" + CANCELLED → RequestStockRelease() → CANCELLED + OrderCancelledEvent (undo reserve)
//	"SUCCESS" + CONFIRMED → idempotent skip (already confirmed)
//	"FAILED"  + PENDING → MarkReservationFailed() → CANCELLED (no event)
//	"FAILED"  + CANCELLED → idempotent skip (already cancelled from another path)
//
// WHY "SUCCESS" + CANCELLED → RequestStockRelease?
//   - Race: customer cancels at T1, stock reserves at T2
//   - Order ถูก cancel แล้ว แต่ product_service ไม่รู้ → stock ถูก reserve ค้างอยู่
//   - ต้องส่ง compensation event เพื่อ release stock ที่ reserve ไปโดยไม่จำเป็น
func (s *orderCommandService) HandleStockResult(ctx context.Context, req *dto.HandleStockResultReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Idempotency check (Inbox Pattern)
		processed, err := s.inboxRepo.HasProcessedMessage(txCtx, req.MessageID)
		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}
		if processed {
			// ถ้าเคย process แล้ว → return nil บอก consumer ว่า "done, don't retry"
			return nil
		}

		// Load Order aggregate (รวม Items เพื่อ RequestStockRelease)
		order, err := s.cmdRepo.GetOrderByID(txCtx, req.OrderID)
		if err != nil {
			if errors.Is(err, domain.ErrOrderNotFound) {
				// Order ไม่มี → อาจถูกลบ (edge case) → log และ skip (idempotent)
				logs.Warn(fmt.Sprintf("HandleStockResult: order %s not found, skipping", req.OrderID))
				return s.saveInboxRecord(txCtx, req.MessageID) // mark processed ป้องกัน retry
			}
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		switch req.Status {
		case "SUCCESS":
			switch order.Status {
			case domain.OrderStatusPending:
				// Normal happy path: reserve สำเร็จ → confirm
				if err := order.ConfirmOrder(); err != nil {
					logs.Error(err)
					return errs.NewUnexpectedError()
				}

			case domain.OrderStatusCancelled:
				// Race condition: order ถูก cancel ก่อน reserve สำเร็จ
				// ต้อง release stock ที่ product_service reserve ไปโดยไม่จำเป็น
				// WHY RequestStockRelease ไม่เปลี่ยน Status?
				//   - Order ยัง CANCELLED อยู่ถูกต้อง แค่ raise compensation event
				order.RequestStockRelease()

			default:
				// CONFIRMED หรือ COMPLETED: idempotent skip (already handled)
				logs.Warn(fmt.Sprintf("HandleStockResult SUCCESS: order %s already in status %s", req.OrderID, order.Status))
			}

		case "FAILED":
			switch order.Status {
			case domain.OrderStatusPending:
				// Reservation ล้มเหลว → cancel order (no stock to release)
				if err := order.MarkReservationFailed(); err != nil {
					logs.Error(err)
					return errs.NewUnexpectedError()
				}

			default:
				// Already cancelled/confirmed from another path → skip
				logs.Warn(fmt.Sprintf("HandleStockResult FAILED: order %s already in status %s", req.OrderID, order.Status))
			}

		default:
			logs.Warn(fmt.Sprintf("HandleStockResult: unknown status %s for order %s", req.Status, req.OrderID))
			// Mark as processed to prevent retry on unknown status
			return s.saveInboxRecord(txCtx, req.MessageID)
		}

		// UpdateOrderStatus (targeted SQL UPDATE)
		if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		// SaveDomainEvents: OrderConfirmedEvent หรือ OrderCancelledEvent (compensation) ถ้ามี
		if err := s.cmdRepo.SaveDomainEvents(txCtx, order); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		// Save inbox record: mark message as processed (ป้องกัน duplicate processing)
		return s.saveInboxRecord(txCtx, req.MessageID)
	})
}

// saveInboxRecord helper ที่รวม logic บันทึก inbox record
func (s *orderCommandService) saveInboxRecord(ctx context.Context, messageID string) error {
	inboxEvt := &domain.InboxEvent{
		ID:         messageID,
		ConsumerID: "order_service_stock", // Consumer group identifier สำหรับ scope idempotency
	}
	if err := s.inboxRepo.SaveProcessedMessage(ctx, inboxEvt); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}
	return nil
}

// ─── USE CASE: Process Payment ───────────────────────────────────────────────

// ProcessPayment ลูกค้าชำระเงิน (CONFIRMED → PAID หรือ AWAITING_PAYMENT)
//
// Flow:
//  1. Validate order (status=CONFIRMED, ownership)
//  2. Check existing payment (idempotency)
//  3. Create payment record (PENDING)
//  4. Call payment gateway (outside TX เพราะเป็น external call)
//  5. Update payment + order status ใน TX เดียว
func (s *orderCommandService) ProcessPayment(ctx context.Context, orderID string, customerID uint, req *dto.ProcessPaymentReq) (*dto.PaymentRes, error) {
	// 1. Load + validate order
	order, err := s.cmdRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			return nil, errs.NewNotFoundError(fmt.Sprintf("order %s not found", orderID))
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	if order.CustomerID != customerID {
		return nil, errs.NewForbiddenError("you do not have permission to pay for this order")
	}

	// อนุญาตทั้ง CONFIRMED (จ่ายครั้งแรก) และ AWAITING_PAYMENT (กดดู QR ซ้ำ / QR เดิมหมดอายุ)
	if order.Status != domain.OrderStatusConfirmed && order.Status != domain.OrderStatusAwaitingPayment {
		return nil, errs.NewValidationError("order is not ready for payment")
	}

	// 2. Idempotency: หา payment เดิมของ order นี้
	existingPayment, err := s.paymentRepo.FindByOrderID(ctx, orderID)
	if err != nil && !errors.Is(err, domain.ErrPaymentNotFound) {
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// 3. เตรียม payment record
	//   - มี payment PENDING อยู่แล้ว (order = AWAITING_PAYMENT) → reuse record เดิมแล้ว re-issue QR
	//     ไม่สร้าง record ใหม่ เพื่อไม่ให้มี payment ซ้ำซ้อนต่อ 1 order (กดปุ่มดู QR ซ้ำได้)
	//   - ยังไม่เคยจ่าย → สร้าง record ใหม่
	var payment *domain.Payment
	if existingPayment != nil {
		if existingPayment.Status == domain.PaymentStatusSuccess {
			return nil, errs.NewConflictError("order already paid")
		}
		// PENDING/FAILED → repay: ใช้ record เดิม (ขอ QR ใหม่)
		payment = existingPayment
		payment.PaymentMethod = req.PaymentMethod
	} else {
		payment = domain.NewPayment(orderID, customerID, order.TotalAmount, "THB", "STUB", req.PaymentMethod)
		if err := s.paymentRepo.Create(ctx, payment); err != nil {
			logs.Error(err)
			return nil, errs.NewUnexpectedError()
		}
	}

	// 4. Call gateway (NOT in TX — external call)
	// WHY call gateway OUTSIDE transaction?
	//   - Stripe/Omise HTTP call ใช้เวลา 1-3 วินาที
	//   - ถ้าอยู่ใน TX → lock DB row ตลอด → blocking queries อื่น → throughput ลด
	//   - Pattern: ทำ external call ก่อน, แล้วค่อยเปิด TX สั้นๆ เพื่ออัปเดต DB
	chargeResult, err := s.paymentGateway.Charge(ctx, &gateway.ChargeRequest{
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     order.TotalAmount,
		Currency:   "THB",
		Token:      req.Token,
		Method:     req.PaymentMethod,
	})
	if err != nil {
		payment.MarkFailed(err.Error())
		_ = s.paymentRepo.UpdateStatus(ctx, payment)
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// 5. Update payment + order ใน TX
	// WHY ต้องอัปเดตทั้ง payment และ order ใน TX เดียวกัน?
	//   - ถ้า payment update แล้ว order ไม่ update (server crash) → inconsistent state
	//   - TX รับประกัน atomicity: สำเร็จทั้งคู่หรือ rollback ทั้งคู่
	switch chargeResult.Status {
	case "SUCCESS":
		payment.MarkSuccess(chargeResult.ChargeID)
		if err := s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
			if err := s.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
				return err
			}
			if err := order.MarkPaid(); err != nil {
				return err
			}
			if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
				return err
			}
			return s.cmdRepo.SaveDomainEvents(txCtx, order)
		}); err != nil {
			logs.Error(err)
			return nil, errs.NewUnexpectedError()
		}

	case "PENDING":
		// Async payment (e.g. PromptPay) — รอ webhook callback
		// WHY ต้องบันทึก ChargeID ใน PENDING case?
		//   - เมื่อ webhook มา → FindByGatewayChargeID(chargeID) → update สำเร็จ
		//   - ถ้าไม่บันทึก ChargeID → webhook callback หา payment record ไม่เจอ
		payment.Status = domain.PaymentStatusPending
		payment.GatewayChargeID = chargeResult.ChargeID
		if err := s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
			if err := s.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
				return err
			}
			// transition CONFIRMED → AWAITING_PAYMENT เฉพาะครั้งแรก
			// ถ้า order = AWAITING_PAYMENT อยู่แล้ว (repay) ไม่ต้อง transition ซ้ำ
			if order.Status == domain.OrderStatusConfirmed {
				if err := order.MarkAwaitingPayment(); err != nil {
					return err
				}
				if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			logs.Error(err)
			return nil, errs.NewUnexpectedError()
		}
		// ClientSecret + QRImageURL ส่งกลับ frontend
		// ClientSecret: สำหรับ 3DS modal (credit card ที่ต้อง confirm ซ้ำ)
		// QRImageURL:   สำหรับ PromptPay — frontend แสดงรูปโดยตรง ไม่ต้อง confirm อีก
		res := mapper.ToPaymentRes(payment)
		res.ClientSecret = chargeResult.ClientSecret
		res.QRImageURL = chargeResult.QRImageURL
		return res, nil

	default: // FAILED
		payment.MarkFailed("charge declined")
		_ = s.paymentRepo.UpdateStatus(ctx, payment)
		return nil, errs.NewValidationError("payment was declined")
	}

	return mapper.ToPaymentRes(payment), nil
}

// ─── USE CASE: Handle Payment Webhook ────────────────────────────────────────

// HandlePaymentWebhook รับ webhook callback จาก payment gateway
// ใช้สำหรับ async payment (PromptPay, bank transfer) ที่ผลลัพธ์มาทีหลัง
func (s *orderCommandService) HandlePaymentWebhook(ctx context.Context, req *dto.PaymentWebhookReq) error {
	// 1. Verify webhook signature
	webhookEvent, err := s.paymentGateway.VerifyWebhook(req.Signature, req.Payload)
	if err != nil {
		return errs.NewValidationError("invalid webhook signature")
	}

	// 2. Find payment by charge ID
	payment, err := s.paymentRepo.FindByGatewayChargeID(ctx, webhookEvent.ChargeID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			logs.Warn(fmt.Sprintf("Webhook: payment not found for charge %s, skipping", webhookEvent.ChargeID))
			return nil
		}
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	// 3. Idempotency: already processed
	// WHY เช็คก่อน process?
	//   - Webhook จาก gateway อาจถูกส่งซ้ำ (at-least-once delivery)
	//   - ถ้าไม่เช็ค → MarkPaid() ถูกเรียกซ้ำ → domain อาจ panic หรือ double-emit event
	if payment.Status == domain.PaymentStatusSuccess || payment.Status == domain.PaymentStatusFailed {
		return nil
	}

	// 4. Load order
	order, err := s.cmdRepo.GetOrderByID(ctx, payment.OrderID)
	if err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	// 5. Process result ใน TX
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		switch webhookEvent.Status {
		case "SUCCESS":
			payment.MarkSuccess(webhookEvent.ChargeID)
			if err := s.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
				logs.Error(err)
				return errs.NewUnexpectedError()
			}
			if err := order.MarkPaid(); err != nil {
				logs.Error(err)
				return errs.NewUnexpectedError()
			}
			if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
				logs.Error(err)
				return errs.NewUnexpectedError()
			}
			return s.cmdRepo.SaveDomainEvents(txCtx, order)

		case "FAILED":
			payment.MarkFailed("gateway reported failure")
			if err := s.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
				logs.Error(err)
				return errs.NewUnexpectedError()
			}
			if err := order.MarkPaymentFailed("gateway reported failure"); err != nil {
				logs.Error(err)
				return errs.NewUnexpectedError()
			}
			if err := s.cmdRepo.UpdateOrderStatus(txCtx, order.ID, order.Status); err != nil {
				logs.Error(err)
				return errs.NewUnexpectedError()
			}
			return s.cmdRepo.SaveDomainEvents(txCtx, order)
		}
		return nil
	})
}

// refundIfPaid ตรวจสอบและ refund ถ้า order ถูกจ่ายเงินไปแล้ว (ก่อน cancel)
//
// WHY ต้อง refund ก่อน cancel?
//   - ถ้า cancel โดยไม่ refund → ลูกค้าเสียเงินแต่ไม่ได้ของ
//   - CancelOrder + AdminCancelOrder ทั้งคู่เรียก refundIfPaid ก่อนเสมอ
//
// WHY check payment.Status == SUCCESS ก่อนเรียก Refund?
//   - payment อาจอยู่ใน PENDING (async) หรือ FAILED → ไม่มีเงินที่ต้อง refund
//   - Refund เฉพาะที่ gateway ยืนยันแล้ว (SUCCESS) เท่านั้น
func (s *orderCommandService) refundIfPaid(ctx context.Context, order *domain.Order) error {
	if order.Status != domain.OrderStatusPaid {
		return nil
	}
	payment, err := s.paymentRepo.FindByOrderID(ctx, order.ID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return nil
		}
		logs.Error(err)
		return errs.NewUnexpectedError()
	}
	if payment.Status != domain.PaymentStatusSuccess {
		return nil
	}
	if err := s.paymentGateway.Refund(ctx, payment.GatewayChargeID, payment.Amount); err != nil {
		logs.Error(fmt.Sprintf("Refund failed for order %s: %v", order.ID, err))
		return errs.NewUnexpectedError()
	}
	payment.MarkRefunded()
	if err := s.paymentRepo.UpdateStatus(ctx, payment); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}
	return nil
}
