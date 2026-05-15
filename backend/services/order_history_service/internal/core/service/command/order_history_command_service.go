// WHAT: OrderHistoryCommandService — event consumer ที่ sync order state ลง MongoDB (write side)
//
// WHY ต้องมี service นี้?
//   - CQRS: write path (order_service → PostgreSQL) แยกจาก read path (order_history → MongoDB)
//   - MongoDB denormalized → ดึงประวัติ order ได้เร็วกว่า JOIN หลาย table ใน PostgreSQL
//
// Inbox Pattern: ทุก Handle* ตรวจ messageID ก่อน + mark processed หลัง
//   - ป้องกัน duplicate เมื่อ Kafka ส่ง message ซ้ำ (At-Least-Once delivery guarantee)
//   - consumerID = "order-history-service" (namespace ไม่ซ้ำกับ catalog-service หรือ consumer อื่น)
//
// Events ที่ handle:
//   OrderCreatedEvent   → Upsert OrderHistory doc (status: "PENDING")
//   OrderConfirmedEvent → UpdateStatus "CONFIRMED"
//   OrderCancelledEvent → MarkCancelled + set cancel_reason
package command

import (
	"context"
	"events"
	"logs"
	"order_history_service/internal/core/domain"
	repo "order_history_service/internal/core/port/repo"
	serviceport "order_history_service/internal/core/port/service"
	"time"
)

const consumerID = "order-history-service"

type orderHistoryCommandService struct {
	writeRepo repo.OrderHistoryWriteRepository
	inboxRepo repo.InboxRepository
}

func NewOrderHistoryCommandService(writeRepo repo.OrderHistoryWriteRepository, inboxRepo repo.InboxRepository) serviceport.OrderHistoryCommandService {
	return &orderHistoryCommandService{
		writeRepo: writeRepo,
		inboxRepo: inboxRepo,
	}
}

// isProcessed ตรวจสอบว่า message นี้ถูก process ไปแล้วหรือยัง (Inbox/Idempotency check)
// WHY ต้องตรวจก่อนทุก event handler?
//   - Kafka guarantee At-Least-Once delivery → message อาจถูกส่งซ้ำ (network retry, consumer rebalance)
//   - ถ้าไม่ตรวจ → Upsert/UpdateStatus ซ้ำ → MongoDB doc อาจ overwrite ข้อมูลที่ถูกต้อง
func (s *orderHistoryCommandService) isProcessed(ctx context.Context, messageID string) (bool, error) {
	return s.inboxRepo.HasProcessed(ctx, messageID, consumerID)
}

// markProcessed บันทึกว่า message นี้ถูก process เรียบร้อยแล้ว
// HOW: insert inbox_messages row → ถ้า DB transaction fail → row ไม่ถูก commit → process ใหม่ได้
func (s *orderHistoryCommandService) markProcessed(ctx context.Context, messageID string) error {
	return s.inboxRepo.MarkProcessed(ctx, &domain.InboxMessage{
		ID:          messageID,
		ConsumerID:  consumerID,
		ProcessedAt: time.Now(),
	})
}

// HandleOrderCreated สร้าง OrderHistory document ใหม่ใน MongoDB
// HOW: event → domain mapping → Upsert (ไม่ใช้ Insert เพื่อ idempotency ในกรณี retry)
// Event → Domain field mapping:
//   evt.OrderID            → order.OrderID
//   evt.CustomerID         → order.CustomerID
//   evt.TotalAmount        → order.TotalAmount
//   evt.Items[].VariantID  → order.Items[].VariantID
//   evt.Items[].UnitPrice  → order.Items[].UnitPrice (snapshot ราคาตอนซื้อ)
//   evt.ShippingAddress    → order.ShippingAddress (embedded struct)
//   (status ตั้งเป็น "PENDING" เสมอ ไม่ใช้ค่าจาก event)
func (s *orderHistoryCommandService) HandleOrderCreated(ctx context.Context, messageID string, evt *events.OrderCreatedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	items := make([]domain.OrderHistoryItem, len(evt.Items))
	for i, item := range evt.Items {
		items[i] = domain.OrderHistoryItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	order := &domain.OrderHistory{
		OrderID:     evt.OrderID,
		CustomerID:  evt.CustomerID,
		Status:      "PENDING",
		TotalAmount: evt.TotalAmount,
		Items:       items,
		ShippingAddress: domain.ShippingAddress{
			FullName:    evt.ShippingAddress.FullName,
			Phone:       evt.ShippingAddress.Phone,
			AddressLine: evt.ShippingAddress.AddressLine,
			SubDistrict: evt.ShippingAddress.SubDistrict,
			District:    evt.ShippingAddress.District,
			Province:    evt.ShippingAddress.Province,
			PostalCode:  evt.ShippingAddress.PostalCode,
		},
		Note:      evt.Note,
		CreatedAt: evt.OccurredAt,
		UpdatedAt: evt.OccurredAt,
	}

	if err := s.writeRepo.Upsert(ctx, order); err != nil {
		return err
	}

	logs.Info("order-history: order created — " + evt.OrderID)
	return s.markProcessed(ctx, messageID)
}

func (s *orderHistoryCommandService) HandleOrderConfirmed(ctx context.Context, messageID string, evt *events.OrderConfirmedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.UpdateStatus(ctx, evt.OrderID, "CONFIRMED"); err != nil {
		return err
	}

	logs.Info("order-history: order confirmed — " + evt.OrderID)
	return s.markProcessed(ctx, messageID)
}

func (s *orderHistoryCommandService) HandleOrderCancelled(ctx context.Context, messageID string, evt *events.OrderCancelledEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.MarkCancelled(ctx, evt.OrderID, evt.Reason); err != nil {
		return err
	}

	logs.Info("order-history: order cancelled — " + evt.OrderID)
	return s.markProcessed(ctx, messageID)
}
