package domain

import "time"

// Category เป็น Domain Object ที่รองรับโครงสร้างแบบ Tree (Self-referencing)
// ParentID เป็น pointer (*uint) ไม่ใช่ uint ธรรมดา เพราะ:
//   - Category ระดับ Root ไม่มี Parent → ParentID = nil
//   - ถ้าใช้ uint ปกติ จะต้องแยกแยะด้วย value 0 ซึ่งอาจชนกับ ID จริงในอนาคต
// Children ถูก load มาเพื่อส่งไปยัง Frontend ในรูปแบบ Tree โดยตรง ไม่ต้องให้ Client ประกอบเอง
type Category struct {
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Slug        string // URL-friendly identifier เช่น "electronics", "men-shoes"
	Description string
	IsActive    bool
	ParentID    *uint      // nil = Root category, non-nil = Sub-category
	Children    []Category // Nested children สำหรับ Tree response
}

// UpdateCategory ใช้ pattern Selective Update
// ทำไมไม่ assign ตรงๆ? เพราะต้องการให้ Service Load aggregate ขึ้นมาก่อน
// แล้วค่อย Apply การเปลี่ยนแปลงผ่าน Method นี้ ซึ่งเป็น Aggregate behavior ที่ถูกต้อง
// IsActive ไม่ได้ทำ guard เพราะ false ก็เป็นค่าที่ถูกต้อง (ต้องการ deactivate ได้)
func (c *Category) UpdateCategory(req *Category) {
	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Slug != "" {
		c.Slug = req.Slug
	}
	if req.Description != "" {
		c.Description = req.Description
	}
	// IsActive ถูก assign ทุกครั้งโดยไม่ตรวจ เพราะ false เป็น valid value ที่หมายถึง "ซ่อน Category นี้"
	c.IsActive = req.IsActive
}
