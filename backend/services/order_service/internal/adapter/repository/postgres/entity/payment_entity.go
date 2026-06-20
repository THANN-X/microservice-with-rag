package entity

import (
	"order_service/internal/core/domain"
	"time"
)

type PaymentEntity struct {
	ID              string     `gorm:"primaryKey;type:varchar(36)"`
	OrderID         string     `gorm:"type:varchar(36);not null;index"`
	CustomerID      uint       `gorm:"not null;index"`
	Amount          float64    `gorm:"type:decimal(12,2);not null"`
	Currency        string     `gorm:"type:varchar(3);not null;default:'THB'"`
	Status          string     `gorm:"type:varchar(50);not null;default:'PENDING';index"`
	Gateway         string     `gorm:"type:varchar(50);not null"`
	GatewayChargeID string     `gorm:"type:varchar(255);index"`
	PaymentMethod   string     `gorm:"type:varchar(50)"`
	PaidAt          *time.Time `gorm:"type:timestamptz"`
	FailedReason    string     `gorm:"type:text"`
	CreatedAt       time.Time  `gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime"`
}

func (PaymentEntity) TableName() string { return "payments" }

func (e *PaymentEntity) ToPaymentDomain() *domain.Payment {
	if e == nil {
		return nil
	}
	return &domain.Payment{
		ID:              e.ID,
		OrderID:         e.OrderID,
		CustomerID:      e.CustomerID,
		Amount:          e.Amount,
		Currency:        e.Currency,
		Status:          domain.PaymentStatus(e.Status),
		Gateway:         e.Gateway,
		GatewayChargeID: e.GatewayChargeID,
		PaymentMethod:   e.PaymentMethod,
		PaidAt:          e.PaidAt,
		FailedReason:    e.FailedReason,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func ToPaymentEntity(p *domain.Payment) *PaymentEntity {
	if p == nil {
		return nil
	}
	return &PaymentEntity{
		ID:              p.ID,
		OrderID:         p.OrderID,
		CustomerID:      p.CustomerID,
		Amount:          p.Amount,
		Currency:        p.Currency,
		Status:          string(p.Status),
		Gateway:         p.Gateway,
		GatewayChargeID: p.GatewayChargeID,
		PaymentMethod:   p.PaymentMethod,
		PaidAt:          p.PaidAt,
		FailedReason:    p.FailedReason,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
