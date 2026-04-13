package dto

import "time"

type CartRes struct {
	CartID    uint          `json:"cart_id"`
	UserID    uint          `json:"user_id"`
	Items     []CartItemRes `json:"items"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type CartItemRes struct {
	VariantID uint      `json:"variant_id"`
	Quantity  int       `json:"quantity"`
	AddedAt   time.Time `json:"added_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
