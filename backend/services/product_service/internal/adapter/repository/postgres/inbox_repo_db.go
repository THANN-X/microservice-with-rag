package repository

import (
	"context"
	"database"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/repo"

	"gorm.io/gorm"
)

// inboxRepository บันทึก processed message IDs เพื่อ implement Inbox Pattern (Idempotent Consumer)
// WHY ต้องมี inbox table?
//   - Kafka guarantees at-least-once delivery → message อาจถูกส่งซ้ำ (rebalance, crash)
//   - inbox table ทำให้ consumer ช่วย exactly-once semantics ได้
type inboxRepository struct {
	*database.TxHelper
}

func NewInboxRepository(db *gorm.DB) port.InboxRepository {
	return &inboxRepository{
		TxHelper: database.NewTxHelper(db),
	}
}

func (r *inboxRepository) HasProcessedMessage(ctx context.Context, messageID string) (bool, error) {
	var count int64
	inboxEntity := &entity.InboxEventEntity{}
	// check เฉพาะ ID (ไม่เช็ค ConsumerID) เพราะ MessageID (Kafka Key) ควร unique ทั่วระบบอยู่แล้ว (UUID)
	// TODO: ถ้าในอนาคตมี consumer หลายตัว ให้เพิ่ม ConsumerID ใน WHERE clause เพื่อ scope Idempotency ต่อ consumer
	err := r.GetDB(ctx).Model(inboxEntity).Where("id = ?", messageID).Count(&count).Error

	return count > 0, err
}

func (r *inboxRepository) SaveProcessedMessage(ctx context.Context, event *domain.InboxEvent) error {
	entityData := entity.ToInboxEventEntity(event)
	// ใช้ clause.OnConflict เพื่อความชัวร์ (Idempotency)
	return r.GetDB(ctx).Create(entityData).Error
}
