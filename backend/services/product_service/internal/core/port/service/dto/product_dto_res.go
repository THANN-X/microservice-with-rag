package service

// RESPONSE DTOs
// เน้น Detail แยกส่วน เพื่อให้ Frontend นำไปใช้ต่อได้ง่าย
type ProductRes struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	IsActive    bool                `json:"is_active"`
	Variants    []ProductVariantRes `json:"variants"`
	Categories  []string            `json:"categories"`
	CreatedBy   uint                `json:"created_by"`
}

type ProductVariantRes struct {
	ID       uint    `json:"variant_id"`
	Sku      string  `json:"sku"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	IsActive bool    `json:"is_active"`
	// ส่งเป็น Object Array เพื่อความยืดหยุ่น
	Options []VariantOptionRes `json:"options"`
}

type VariantOptionRes struct {
	Name  string `json:"name"`  // "Color"
	Value string `json:"value"` // "Red"
}

type ProductListRes struct {
	Items      []ProductRes `json:"items"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}
