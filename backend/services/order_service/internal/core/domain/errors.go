// WHAT: Domain-level sentinel errors สำหรับ order_service
// WHY: แยก error ที่ domain กำหนด ออกจาก infrastructure error
//   - Service layer ใช้ errors.Is() เพื่อแปล domain error → HTTP error code ที่เหมาะสม
//   - ป้องกัน magic string ("order not found") กระจายในโค้ด
package domain

import "errors"

var (
	// Aggregate not found
	ErrOrderNotFound = errors.New("order not found")

	// State transition violations (Invariant #5)
	ErrInvalidOrderTransition    = errors.New("invalid order state transition")
	ErrOrderAlreadyCancelled     = errors.New("order is already cancelled")
	ErrCannotCancelCompletedOrder = errors.New("cannot cancel a completed order")

	// Factory method validation (Invariants #1-#3)
	ErrInvalidCustomer  = errors.New("invalid customer ID")
	ErrEmptyOrderItems  = errors.New("order must have at least one item")
	ErrInvalidQuantity  = errors.New("item quantity must be greater than 0")
	ErrInvalidPrice     = errors.New("item unit price must be greater than 0")

	// Access control
	ErrUnauthorized = errors.New("you do not own this order")

	// Generic DB-level
	ErrNoDataModified = errors.New("no data was modified")
	ErrInternal       = errors.New("internal error")

	// Payment
	ErrPaymentAlreadyExists = errors.New("payment already exists for this order")
	ErrPaymentNotFound      = errors.New("payment not found")
)
