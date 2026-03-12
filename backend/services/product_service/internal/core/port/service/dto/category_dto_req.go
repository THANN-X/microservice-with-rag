package service

type CreateCategoryReq struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Slug        string `json:"slug" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=255"`
	IsActive    bool   `json:"is_active"`
	ParentID    *uint  `json:"parent_id"`
}

type UpdateCategoryReq struct {
	CategoryID  uint   `json:"category_id" validate:"required"`
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Slug        string `json:"slug" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=255"`
	IsActive    bool   `json:"is_active"`
	ParentID    *uint  `json:"parent_id"`
}

// SetCategoryActiveReq ใช้ Toggle active/inactive ของ Category
// แยกจาก UpdateCategoryReq เพราะ:
//   - Active toggle ควร explicit ไม่ปนกับการแก้ชื่อ/slug
//   - Route PATCH /categories/admin/:id/active อ่านออกทันทีว่าทำอะไร
type SetCategoryActiveReq struct {
	CategoryID uint `json:"category_id"` // ถูก populate จาก URL param โดย Handler
	IsActive   bool `json:"is_active"`
}
