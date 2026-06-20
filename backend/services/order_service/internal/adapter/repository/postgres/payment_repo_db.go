// WHAT: PostgreSQL implementation ของ PaymentRepository interface
//
// WHY embed *database.TxHelper แทนเก็บ *gorm.DB ตรงๆ?
//   - TxHelper.GetDB(ctx) จะคืน TX ที่อยู่ใน context ถ้ามี หรือ db ปกติถ้าไม่มี
//   - ทำให้ repo ทำงานได้ทั้งแบบ standalone และแบบอยู่ใน RunInTx() โดยไม่ต้องเปลี่ยนโค้ด
//   - Pattern เดียวกับ orderRepository (consistency ของ codebase)
package repository

import (
	"context"
	"database"
	"errors"
	"order_service/internal/adapter/repository/postgres/entity"
	"order_service/internal/core/domain"
	port "order_service/internal/core/port/repo"

	"gorm.io/gorm"
)

type paymentRepository struct {
	*database.TxHelper
}

func NewPaymentRepository(db *gorm.DB) port.PaymentRepository {
	return &paymentRepository{
		TxHelper: database.NewTxHelper(db),
	}
}

func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	e := entity.ToPaymentEntity(payment)
	return r.GetDB(ctx).Create(e).Error
}

func (r *paymentRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	var e entity.PaymentEntity
	// WHY Order("created_at DESC")?
	//   - Order เดียวอาจมีหลาย Payment record (retry หลังจาก failed, หรือ refund สร้าง record ใหม่)
	//   - เราต้องการ payment ล่าสุด เช่น เช็ค idempotency หรือหา charge ID สำหรับ refund
	err := r.GetDB(ctx).Where("order_id = ?", orderID).Order("created_at DESC").First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return e.ToPaymentDomain(), nil
}

func (r *paymentRepository) FindByGatewayChargeID(ctx context.Context, chargeID string) (*domain.Payment, error) {
	var e entity.PaymentEntity
	err := r.GetDB(ctx).Where("gateway_charge_id = ?", chargeID).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return e.ToPaymentDomain(), nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, payment *domain.Payment) error {
	// WHY ใช้ map[string]interface{} แทน .Save(entity)?
	//   - .Save() จะ UPDATE ทุก field → อาจเขียนทับค่าที่ goroutine อื่น update ไปพร้อมกัน (race condition)
	//   - targeted UPDATE เฉพาะ fields ที่ status เปลี่ยน → atomic และปลอดภัยกว่า
	//   - GORM ยังมี bug ที่ .Save() กับ zero-value fields บางตัว (เช่น PaidAt=nil) อาจไม่ทำงานถูกต้อง
	return r.GetDB(ctx).Model(&entity.PaymentEntity{}).
		Where("id = ?", payment.ID).
		Updates(map[string]interface{}{
			"status":            string(payment.Status),
			"gateway_charge_id": payment.GatewayChargeID,
			"paid_at":           payment.PaidAt,
			"failed_reason":     payment.FailedReason,
			"updated_at":        payment.UpdatedAt,
		}).Error
}
