// WHAT: GORM implementation ของ OutboxRepository
// ใช้โดย OutboxProcessor background worker เท่านั้น (ไม่ใช่ใน business TX)
// WHY แยก OutboxRepository จาก OrderCommandRepository?
//   - OutboxProcessor มี lifecycle ของตัวเอง (polling, retry, dead letter)
//   - ไม่เกี่ยวกับ Order aggregate → แยก interface ทำให้ dependency ชัดเจน
package repository

import (
	"context"
	"database"
	"order_service/internal/adapter/repository/postgres/entity"
	"order_service/internal/core/domain"
	port "order_service/internal/core/port/repo"
	"time"

	"gorm.io/gorm"
)

type outboxRepository struct {
	*database.TxHelper
}

func NewOutboxRepository(db *gorm.DB) port.OutboxRepository {
	return &outboxRepository{
		TxHelper: database.NewTxHelper(db),
	}
}

// Save บันทึก outbox event ใหม่
// เรียกจาก SaveDomainEvents ภายใน TX context → ใช้ TxHelper.GetDB(ctx) เพื่อ TX awareness
func (r *outboxRepository) Save(ctx context.Context, event *domain.OutboxEvent) error {
	entityData := entity.ToOutboxEventEntity(event)
	return r.GetDB(ctx).Create(entityData).Error
}

// GetUnsentMessages ดึง PENDING batch เรียง created_at ASC (FIFO)
// WHY FIFO?
//   - Events มี ordering: Created ก่อน Cancelled → ถ้าส่งผิดลำดับ downstream อาจ misinterpret
//   - Kafka Partition Key (AggregateID) การันตี per-aggregate ordering อีกชั้น
// WHY batch size 20 (ไม่ unlimited)?
//   - ป้องกัน memory spike ถ้า Kafka down นานแล้ว PENDING สะสม
//   - Trade-off: latency สูงขึ้นเล็กน้อย แต่ memory safe
func (r *outboxRepository) GetUnsentMessages(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	var entityMessages []*entity.OutboxEventEntity

	err := r.GetDB(ctx).
		Where("status = ?", "PENDING").
		Order("created_at ASC").
		Limit(limit).
		Find(&entityMessages).Error

	if err != nil {
		return nil, err
	}

	messages := make([]*domain.OutboxEvent, len(entityMessages))
	for i, e := range entityMessages {
		messages[i] = e.ToOutboxEventDomain()
	}

	return messages, nil
}

// MarkAsSent อัปเดต status=SENT + sent_at แบบ atomic
func (r *outboxRepository) MarkAsSent(ctx context.Context, id string) error {
	now := time.Now()
	// WHY Updates + map แทน 2 statements?
	//   - map ไม่ถูก GORM skip zero-value (ต่างจาก struct update)
	//   - Atomic: status + sent_at อัปเดตพร้อมกัน
	return r.GetDB(ctx).Model(&entity.OutboxEventEntity{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  domain.OutboxStatusSent,
			"sent_at": &now,
		}).Error
}

// IncrementRetryCount เพิ่ม retry_count แบบ atomic SQL expression
// WHY ไม่ Load-add-Save?
//   - ถ้า processor รัน 2+ instances พร้อมกัน → Load เห็นค่าเดิม → retry_count ผิด
//   - SQL `SET retry_count = retry_count + 1` เป็น atomic ที่ DB engine level
func (r *outboxRepository) IncrementRetryCount(ctx context.Context, id string) error {
	return r.GetDB(ctx).Model(&entity.OutboxEventEntity{}).
		Where("id = ?", id).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error
}

// MarkAsFailed ตั้ง status=FAILED + error_message (Dead Letter Queue simulation)
// WHY เก็บ error message?
//   - ช่วย debug ว่า event ล้มเพราะอะไร (e.g. Kafka unreachable, serialization error)
//   - TODO: alert monitoring ถ้ามี FAILED events สะสม (e.g. PagerDuty, Grafana)
func (r *outboxRepository) MarkAsFailed(ctx context.Context, id, errMsg string) error {
	return r.GetDB(ctx).Model(&entity.OutboxEventEntity{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        domain.OutboxStatusFailed,
			"error_message": errMsg,
		}).Error
}
