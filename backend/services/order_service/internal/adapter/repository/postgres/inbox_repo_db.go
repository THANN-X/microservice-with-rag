// WHAT: GORM implementation ของ InboxRepository (Idempotent Consumer)
// ตรวจสอบ MessageID ก่อนประมวลผล → ป้องกัน duplicate handling จาก Kafka at-least-once
package repository

import (
	"context"
	"database"
	"order_service/internal/adapter/repository/postgres/entity"
	"order_service/internal/core/domain"
	port "order_service/internal/core/port/repo"

	"gorm.io/gorm"
)

type inboxRepository struct {
	*database.TxHelper
}

func NewInboxRepository(db *gorm.DB) port.InboxRepository {
	return &inboxRepository{
		TxHelper: database.NewTxHelper(db),
	}
}

// HasProcessedMessage ตรวจสอบว่า MessageID นี้เคยถูก process แล้วหรือยัง
// WHY COUNT แทน First + error check?
//   - COUNT ไม่ error ถ้า record ไม่มี (First คืน ErrRecordNotFound)
//   - เร็วกว่า (DB ไม่ต้อง fetch row data, แค่นับ)
func (r *inboxRepository) HasProcessedMessage(ctx context.Context, messageID string) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entity.InboxEventEntity{}).
		Where("id = ?", messageID).
		Count(&count).Error

	return count > 0, err
}

// SaveProcessedMessage บันทึก inbox record หลังจาก process สำเร็จ
// WHY ต้องเรียกใน TX เดียวกับ business operation?
//   - ถ้า business op สำเร็จแต่ SaveProcessedMessage ล้ม → message จะถูก process ซ้ำ
//   - ถ้าทั้งคู่อยู่ใน TX เดียว → ถ้า TX fail ทั้งคู่ rollback (idempotent retry ok)
//   - TxHelper.GetDB(ctx) จะใช้ TX context โดยอัตโนมัติ
func (r *inboxRepository) SaveProcessedMessage(ctx context.Context, event *domain.InboxEvent) error {
	entityData := entity.ToInboxEventEntity(event)
	return r.GetDB(ctx).Create(entityData).Error
}
