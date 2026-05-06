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
	VariantID   uint      `json:"variant_id"`
	Quantity    int       `json:"quantity"`
	ProductName string    `json:"product_name,omitempty"`
	VariantName string    `json:"variant_name,omitempty"`
	Price       float64   `json:"price,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	AddedAt     time.Time `json:"added_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
