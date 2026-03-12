package repository

import (
	"context"
	"database"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/repo"
	"time"

	"gorm.io/gorm"
)

// outboxRepository ห้อม outbox table เพื่อ implement Transactional Outbox Pattern
// WHY ไม่คุยกับ Kafka โดยตรง?
//   - ถ้า save DB สำเร็จแต่ส่ง Kafka ล้ม → event หาย (data inconsistency)
//   - Outbox Pattern: save event ลง DB เดียวกับ business data ใน TX เดียวกัน
//   - OutboxProcessor อ่าน outbox table แล้วค่อยๆ ส่งไป Kafka (at-least-once)
type outboxRepository struct {
	*database.TxHelper
}

func NewOutboxRepository(db *gorm.DB) port.OutboxRepository {
	return &outboxRepository{
		TxHelper: database.NewTxHelper(db),
	}
}

func (r *outboxRepository) Save(ctx context.Context, event *domain.OutboxEvent) error {
	entityData := entity.ToOutboxEventEntity(event)

	return r.GetDB(ctx).Create(entityData).Error
}

// GetUnsentMessages ดึง PENDING messages เรียง FIFO batch limit รั้งละ `limit` records
// WHY FIFO (ORDER BY created_at ASC)?
//   - รับประกัน ordering ของ events: event เก่า ถูกส่งก่อนเสมอ (e.g. Created ก่อน Updated)
//   - Kafka Partition Key การันตี per-aggregate ordering อีกชั้น
// WHY batch size ไม่ unlimited?
//   - ป้องกัน memory spike ถ้ามี PENDING จำนวนมาก (e.g. Kafka down หลายชั่วโมง)
func (r *outboxRepository) GetUnsentMessages(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	var entityMessages []*entity.OutboxEventEntity
	// ดึงข้อมูลเรียงตามเวลาเก่าสุดก่อน (FIFO)
	err := r.GetDB(ctx).Where("status = ?", "PENDING").Order("created_at asc").Limit(limit).Find(&entityMessages).Error

	if err != nil {
		return nil, err
	}

	var messages []*domain.OutboxEvent
	for _, e := range entityMessages {
		messages = append(messages, e.ToOutboxEventDomain())
	}

	return messages, nil
}

func (r *outboxRepository) MarkAsSent(ctx context.Context, id string) error {
	now := time.Now()
	// Update status และ sent_at พร้อมกันใน statement เดียว
	// WHY Updates + map แทน 2 statements?
	//   - Atomic: หลีก race condition ระหว่างการ update 2 fields แยก statement
	//   - GORM map ไม่ถูก skip zero-value (ต่างจาก struct)
	return r.GetDB(ctx).Model(&entity.OutboxEventEntity{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  domain.OutboxStatusSent,
			"sent_at": &now}).Error
}

func (r *outboxRepository) IncrementRetryCount(ctx context.Context, id string) error {
	// Atomic UPDATE: retry_count = retry_count + 1 ผ่าน gorm.Expr
	// WHY ไม่ใช้ Load-add-Save?
	//   - ถ้า OutboxProcessor รัน 2 instance (horizontal scale) Load พร้อมกัน → retry_count ผิด
	//   - SQL `SET retry_count = retry_count + 1` เป็น atomic โดย DB engine
	return r.GetDB(ctx).Model(&entity.OutboxEventEntity{}).Where("id = ?", id).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error
}

func (r *outboxRepository) MarkAsFailed(ctx context.Context, id string, errMsg string) error {
	// Dead Letter: status = FAILED + error_message เพื่อ debug ว่าล้มเพราะอะไร
	// หลังจากนี้ OutboxProcessor จะไม่หยิบ message นี้มาส่งอีก (status != PENDING)
	// TODO: ส่ง alert (Slack/PagerDuty) เมื่อมี message เข้า FAILED เพื่อ operator มา inspect
	return r.GetDB(ctx).Model(&entity.OutboxEventEntity{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        domain.OutboxStatusFailed,
			"error_message": errMsg}).Error
}

// Implement Delete (เมื่อส่งเสร็จแล้วลบทิ้ง)
// func (r *outboxRepository) Delete(ctx context.Context, id string) error {
// 	return r.GetDB(ctx).Delete(&domain.OutboxEvent{}, id).Error
// }
