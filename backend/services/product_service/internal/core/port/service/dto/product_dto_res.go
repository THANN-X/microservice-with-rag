package service

// RESPONSE DTOs
// เน้น Detail แยกส่วน เพื่อให้ Frontend นำไปใช้ต่อได้ง่าย
type ProductRes struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	ImageUrls   []string            `json:"image_urls"`
	IsActive    bool                `json:"is_active"`
	Variants    []ProductVariantRes `json:"variants"`
	// WHY: ใช้ ProductCategoryRes แทน []string + []uint แยกกัน
	//      เพื่อให้ ID และ Name อยู่คู่กันเสมอ ไม่เกิด index mismatch
	Categories []ProductCategoryRes `json:"categories"`
	CreatedBy  uint                 `json:"created_by"`
}

// WHAT: ProductCategoryRes ใช้แสดง category ที่ product นี้อยู่
// WHY: ProductRes ต้องการแค่ ID กับ Name เท่านั้น ไม่ต้องการ slug, children หรือ timestamps
type ProductCategoryRes struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ProductVariantRes struct {
	ID        uint    `json:"id"`
	Sku       string  `json:"sku"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	IsActive  bool    `json:"is_active"`
	ImageUrls []string `json:"image_urls"`
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
