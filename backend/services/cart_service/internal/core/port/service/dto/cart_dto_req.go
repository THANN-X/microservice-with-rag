package dto

// AddCartItemReq — POST /cart/items
// ProductName/VariantName/Price/ImageURL เป็น optional metadata ที่ frontend ส่งมา
// เพื่อ denormalize ไว้ใน cart_items ณ เวลาที่ add (ไม่ต้อง join product service ตอน read)
type AddCartItemReq struct {
	VariantID   uint    `json:"variant_id"   validate:"required"`
	Quantity    int     `json:"quantity"     validate:"required,min=1"`
	ProductName string  `json:"product_name"`
	VariantName string  `json:"variant_name"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
}

// UpdateCartItemReq — PUT /cart/items/:variantId
type UpdateCartItemReq struct {
	VariantID uint `json:"variant_id"`
	Quantity  int  `json:"quantity" validate:"min=0"`
}
