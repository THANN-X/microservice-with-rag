package database

import (
	"context"

	"gorm.io/gorm"
)

type contextkey string

const txKey contextkey = "tx_db"

type TxHelper struct {
	db *gorm.DB
}

func NewTxHelper(db *gorm.DB) *TxHelper {
	return &TxHelper{db: db}
}

func (h *TxHelper) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return h.db.WithContext(ctx)
}

func (h *TxHelper) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		txCtxx := context.WithValue(ctx, txKey, tx)
		return fn(txCtxx)
	})
}
