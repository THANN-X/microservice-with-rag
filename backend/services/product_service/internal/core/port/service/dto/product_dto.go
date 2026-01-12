package service

// internal/core/dto/product_dto.go

// --- REQUEST (ตอน Frontend ส่งมาสร้าง) ---
// เน้น ID เพื่อความไวในการ Save ลง DB
type CreateProductReq struct {
	Name        string             `json:"name" validate:"required"`
	Description string             `json:"description"`
	CategoryID  []uint             `json:"category_ids" validate:"required,min=1"`
	Variants    []CreateVariantReq `json:"variants" validate:"dive"`
}

type CreateVariantReq struct {
	Sku   string  `json:"sku" validate:"required"`
	Price float64 `json:"price" validate:"gt=0"`
	Stock int     `json:"stock" validate:"gte=0"`
	// ส่งมาแค่ ID พอ เช่น [101, 205] (101=Red, 205=XL)
	AttributeValueIDs []uint `json:"attribute_value_ids" validate:"required,min=1"`
}

// --- RESPONSE (ตอนส่งกลับไปให้ Frontend โชว์) ---
// เน้น Detail แยกส่วน เพื่อให้ Frontend จัดหน้าตาสวยๆ ได้
type ProductRes struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Variants    []ProductVariantRes `json:"variants"`
	Categories  []string            `json:"categories"`
}

type ProductVariantRes struct {
	ID    uint    `json:"variant_id"`
	Sku   string  `json:"sku"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
	// ส่งเป็น Object Array เพื่อความยืดหยุ่น
	Options []VariantOptionRes `json:"options"`
}

type VariantOptionRes struct {
	Name  string `json:"name"`  // "Color"
	Value string `json:"value"` // "Red"
}
