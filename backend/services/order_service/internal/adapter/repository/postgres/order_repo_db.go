// WHAT: GORM implementations ของ OrderCommandRepository และ OrderQueryRepository
//
// WHY struct เดียว implement ทั้ง Command และ Query?
//   - ปัจจุบัน order_service ยังไม่จำเป็นต้องแยก read replica
//   - Interface แยกแล้ว → ถ้าต้องการ scale แค่ implement QueryRepository ใหม่ (ไม่กระทบ Command)
//
// WHY embed *database.TxHelper?
//   - TxHelper.GetDB(ctx) คืน TX DB ถ้า ctx มี TX อยู่ มิฉะนั้น baseline DB
//   - ทุก repo call จึงใช้ TX โดยอัตโนมัติโดยไม่ต้องส่ง *gorm.DB ไปทุกที่
//   - RunInTx สร้าง TX ใหม่และเก็บใน ctx → ส่ง txCtx ให้ทุก call ภายใน TX
package repository

import (
	"context"
	"database"
	"encoding/json"
	"errors"
	"order_service/internal/adapter/repository/postgres/entity"
	"order_service/internal/core/domain"
	port "order_service/internal/core/port/repo"
	"time"

	"gorm.io/gorm"
)

type orderRepository struct {
	*database.TxHelper
}

// NewOrderRepository คืน Command และ Query repository จาก struct เดียว
// WHY คืน 2 interfaces จาก 1 struct?
//   - DI container (main.go) ต้องการแยก role → ส่ง interface ที่ถูกต้องไปยัง service ที่ถูกต้อง
//   - Struct เดียวกันทำงานได้ทั้ง 2 roles → ไม่ต้อง duplicate DB connection
func NewOrderRepository(db *gorm.DB) (port.OrderCommandRepository, port.OrderQueryRepository) {
	repo := &orderRepository{
		TxHelper: database.NewTxHelper(db),
	}
	return repo, repo
}

// ─── OrderCommandRepository Implementation ────────────────────────────────────

// CreateOrder INSERT Order + Items ในครั้งเดียว
// WHY ใช้ Create กับ *entity มีทั้ง Items?
//   - GORM cascade create: ถ้า Items มี foreignKey ถูกต้อง GORM จะ INSERT items ด้วย
//   - Atomic: Order + Items ใน TX เดียว (จาก RunInTx ใน service layer)
func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	orderEntity := entity.ToOrderEntity(order)

	if err := r.GetDB(ctx).Create(orderEntity).Error; err != nil {
		return err
	}

	// Sync timestamps กลับไป domain (เผื่อ service ต้องการ)
	order.CreatedAt = orderEntity.CreatedAt
	order.UpdatedAt = orderEntity.UpdatedAt

	return nil
}

// GetOrderByID โหลด Order พร้อม Items สำหรับ command operations
// WHY Preload Items?
//   - Cancel() ต้องการ Items เพื่อสร้าง OrderCancelledEvent.Items
//   - ConfirmOrder() ไม่ต้องการ Items แต่ safe กว่าถ้า load ทุกครั้ง
//
// TODO: ถ้า performance เป็นปัญหา ทำ GetOrderStatusOnly() แยก สำหรับ ConfirmOrder path
func (r *orderRepository) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	orderEntity := &entity.OrderEntity{}

	err := r.GetDB(ctx).Preload("Items").First(orderEntity, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}

	return orderEntity.ToOrderDomain(), nil
}

// UpdateOrderStatus ทำ targeted UPDATE status field เท่านั้น
// WHY targeted UPDATE แทน full Save?
//   - ป้องกัน overwrite fields อื่นที่อาจถูกแก้โดย concurrent process
//   - SQL: UPDATE orders SET status=? WHERE id=?
//   - SQL injection prevention: Status เป็น typed constant (ไม่ใช่ raw string จาก user)
func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	result := r.GetDB(ctx).Model(&entity.OrderEntity{}).
		Where("id = ?", orderID).
		Update("status", string(status))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrOrderNotFound
	}

	return nil
}

// SaveDomainEvents บันทึก domain events จาก Order aggregate ลง outbox table
// WHY จบภายใน Repository (ไม่ผ่าน OutboxRepository.Save)?
//   - ต้องการ TX context เดียวกับ CreateOrder/UpdateOrderStatus
//   - ถ้าเรียกผ่าน OutboxRepository แยก → ต้องส่ง txCtx ผ่านไปด้วย (ทำได้แต่ coupling มากขึ้น)
//   - Pattern นี้เดียวกับ product_service.SaveDomainEvents → consistent conventions
//
// WHY PopDomainEvents ก่อน loop?
//   - Pop = ดึง + clear ในครั้งเดียว → ป้องกัน double-save ถ้า method ถูกเรียกซ้ำ
func (r *orderRepository) SaveDomainEvents(ctx context.Context, order *domain.Order) error {
	evts := order.PopDomainEvents()
	if len(evts) == 0 {
		return nil
	}

	for _, evt := range evts {
		payloadBytes, err := json.Marshal(evt)
		if err != nil {
			return err
		}

		// WHY AggregateID = order.ID เสมอ (ทุก event type)?
		//   - AggregateID ถูกใช้เป็น Kafka message key → ทุก event ของ order เดียวกันลง partition
		//     เดียวกัน → consumer เห็นตามลำดับจริง (CREATED ก่อน CONFIRMED ก่อน CANCELLED)
		//   - เดิมมี switch ที่ตั้งใจให้ cancel event ใช้ key "cancel-<id>" แต่ case เทียบกับ
		//     "OrderCancelled" ขณะที่ EventName() คืน "ORDER_CANCELLED" → เงื่อนไขไม่เคยเป็นจริง
		//     พฤติกรรมจริงคือใช้ order.ID มาตลอด จึงลบ switch ทิ้งให้ตรงกับความจริง
		//   - เจตนาเดิม (กัน inbox ที่ปลายทาง block cancel event ที่ควรผ่าน) ตอนนี้ถูกแทนด้วย
		//     header "EventID" = outbox UUID ซึ่ง unique ต่อ event อยู่แล้ว consumer ทุกตัวใช้ค่านั้น
		//     เป็น inbox key ไม่ได้ใช้ Kafka key → ไม่มีอะไรหายไปจากการลบ switch
		//   - ถ้าจะรื้อฟื้นเจตนาเดิมต้องระวัง: เปลี่ยน key = เปลี่ยน partition = เสีย ordering
		outboxMsg := domain.NewOutboxMessage(
			"order.events",
			order.ID,
			"ORDER",
			evt.EventName(),
			string(payloadBytes),
		)

		outboxEntity := entity.ToOutboxEventEntity(outboxMsg)
		if err := r.GetDB(ctx).Create(outboxEntity).Error; err != nil {
			return err
		}
	}

	return nil
}

// ─── OrderQueryRepository Implementation ──────────────────────────────────────

// FindByID ดึง Order เดี่ยว พร้อม Items (สำหรับ display)
func (r *orderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	orderEntity := &entity.OrderEntity{}

	err := r.GetDB(ctx).Preload("Items").First(orderEntity, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}

	return orderEntity.ToOrderDomain(), nil
}

// NOTE: FindByCustomerID / FindAll ถูกย้ายไป order_history_service (CQRS read side)

// FindExpiredPaymentOrders ค้นหา orders ที่อยู่ใน AWAITING_PAYMENT นานเกิน timeout
func (r *orderRepository) FindExpiredPaymentOrders(ctx context.Context, timeout time.Duration) ([]*domain.Order, error) {
	var entities []entity.OrderEntity
	cutoff := time.Now().Add(-timeout)

	err := r.GetDB(ctx).
		Preload("Items").
		Where("status = ? AND updated_at < ?", string(domain.OrderStatusAwaitingPayment), cutoff).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	orders := make([]*domain.Order, len(entities))
	for i, e := range entities {
		orders[i] = e.ToOrderDomain()
	}

	return orders, nil
}
