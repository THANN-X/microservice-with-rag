package dto

// AddCartItemReq — POST /cart/items
type AddCartItemReq struct {
	VariantID uint `json:"variant_id" validate:"required"`
	Quantity  int  `json:"quantity"   validate:"required,min=1"`
}

// UpdateCartItemReq — PUT /cart/items/:variantId
type UpdateCartItemReq struct {
	VariantID uint `json:"variant_id"`
	Quantity  int  `json:"quantity" validate:"min=0"`
}
