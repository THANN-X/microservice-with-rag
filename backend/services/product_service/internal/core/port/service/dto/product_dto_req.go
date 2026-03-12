package service

// Create Product Request
// Command Requests
// เน้น ID เพื่อความไวในการ Save ลง DB
type CreateProductReq struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description" validate:"required,min=1,max=255"`
	ImageURLs   []string `json:"image_urls" validate:"required,min=1,dive,url"`

	Variants []CreateVariantReq `json:"variants" validate:"dive"`
	// e.g. [1, 5, 10] (1=Electronics, 5=Gadgets, 10=Accessories)
	CategoryIDs []uint `json:"category_ids" validate:"required,min=1"`
}

type CreateVariantReq struct {
	Sku   string  `json:"sku" validate:"required"`
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"gt=0"`
	Stock int     `json:"stock" validate:"gt=0"`
	// e.g. [101, 205] (101=Red, 205=XL)
	AttributeValueIDs []uint `json:"attribute_value_ids" validate:"required,min=1"`
}

// New: Request สำหรับเพิ่ม Variant ใหม่ในสินค้าเดิม
type AddVariantReq struct {
	ProductID         uint    `json:"product_id" validate:"required"`
	Sku               string  `json:"sku" validate:"required"`
	Name              string  `json:"name" validate:"required"` // e.g. "Red / XL"
	Price             float64 `json:"price" validate:"gt=0"`
	Stock             int     `json:"stock" validate:"gt=0"`
	AttributeValueIDs []uint  `json:"attribute_value_ids" validate:"required,min=1"`
}

// New: Request สำหรับ Admin ปรับสต็อก (Stock Take / ของเสีย)
type AdjustStockReq struct {
	ProductID uint   `json:"product_id" validate:"required"`
	VariantID uint   `json:"variant_id" validate:"required"`
	NewStock  int    `json:"new_stock" validate:"gt=0"`
	Reason    string `json:"reason" validate:"required,min=1,max=255"` // e.g. "Damage", "Found"
}

type ListProductReq struct {
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
	Search   string `query:"search"`
	Category uint   `query:"category"`
	IsActive *bool  `query:"is_active"`                                                      // pointer รับ nil(all), true(active), false(inactive)
	SortBy   string `query:"sort_by" validate:"omitempty,oneof=created_at name price stock"` // e.g., "created_at", "name"
	Order    string `query:"order" validate:"omitempty,oneof=asc desc"`                      // e.g., "asc", "desc"
}

// Update General Info Request
type UpdateProductGeneralInfoReq struct {
	ProductID   uint   `json:"product_id" validate:"required"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"required,min=1,max=255"`
	CategoryIDs []uint `json:"category_ids" validate:"required,min=1"`
}

// Update Price Request
type UpdateVariantPriceReq struct {
	ProductID uint    `json:"product_id" validate:"required"` // ต้องมีเพื่อ Load Aggregate Root
	VariantID uint    `json:"variant_id" validate:"required"`
	NewPrice  float64 `json:"new_price" validate:"gt=0"`
}

// SetProductActiveReq ใช้สำหรับ Toggle active/inactive ของสินค้า
// แยกออกจาก UpdateProductGeneralInfoReq เพราะ:
//   - Operation นี้ใช้บ่อย (bulk publish/unpublish) ไม่ต้องการ full body
//   - ช่วยให้ Route ชัดเจน: PATCH /products/admin/:id/active
type SetProductActiveReq struct {
	ProductID uint `json:"product_id"` // ถูก populate จาก URL param โดย Handler
	IsActive  bool `json:"is_active"`
}

// SetVariantActiveReq ใช้ Toggle active/inactive ของ Variant เฉพาะตัว
// ตรวจสอบ variant ownership ผ่าน ProductID ใน Service Layer
type SetVariantActiveReq struct {
	ProductID uint `json:"product_id"` // ถูก populate จาก URL param โดย Handler
	VariantID uint `json:"variant_id"` // ถูก populate จาก URL param โดย Handler
	IsActive  bool `json:"is_active"`
}

// Event Payload ที่ได้รับจาก Message Broker (เช่น Order Service ส่งมา)
type ReserveStockReq struct {
	MessageID string             `json:"-"` // รับจาก Header/Metadata ของ Kafka/RabbitMQ
	OrderID   string             `json:"order_id"`
	Items     []ReserveStockItem `json:"items"`
}

type ReserveStockItem struct {
	VariantID uint `json:"variant_id"`
	Qty       int  `json:"qty"`
}
