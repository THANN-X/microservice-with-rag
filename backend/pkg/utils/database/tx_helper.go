package database

import (
	"context"

	"gorm.io/gorm"
)

// contextkey is a type for context keys used in this package.
type contextkey string

// txKey is the context key for storing the transaction DB instance.
const txKey contextkey = "tx_db"

type TxHelper struct {
	db *gorm.DB
}

func NewTxHelper(db *gorm.DB) *TxHelper {
	return &TxHelper{db: db}
}

// GetDB retrieves the current transaction DB from the context if it exists;
func (h *TxHelper) GetDB(ctx context.Context) *gorm.DB {
	// Check if a transaction DB is stored in the context
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	// Otherwise, return the base DB
	return h.db.WithContext(ctx)
}

// RunInTx runs the provided function within a database transaction.
func (h *TxHelper) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	// Start a new transaction
	return h.db.Transaction(func(tx *gorm.DB) error {
		// Create a new context with the transaction DB
		txCtx := context.WithValue(ctx, txKey, tx)
		// Execute the provided function with the transaction context
		return fn(txCtx)
	})
}
