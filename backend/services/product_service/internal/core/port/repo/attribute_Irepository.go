package port

import (
	"context"
	"product_service/internal/core/domain"
)

// AttributeCommandRepository จัดการ Write operations สำหรับ Attribute และ Value
// ทำไมถึงรวม Attribute และ AttributeValue ไว้ใน interface เดียว?
//   - ทั้งคู่เป็น operations ของ Admin จัดการ Attribute catalog
//   - ถ้าแยก interface จะทำให้ main.go wire repositories เพิ่มโดยไม่จำเป็น
//   - AttributeValue ไม่มี lifecycle ของตัวเอง มันขึ้นอยู่กับ Attribute เสมอ
type AttributeCommandRepository interface {
	CreateAttribute(ctx context.Context, attr *domain.Attribute) error
	UpdateAttribute(ctx context.Context, attr *domain.Attribute) error
	// DeleteAttribute cascade ลบ AttributeValues ด้วยอัตโนมัติ (ผ่าน OnDelete:CASCADE ใน entity)
	DeleteAttribute(ctx context.Context, id uint) error

	CreateAttributeValue(ctx context.Context, val *domain.AttributeValue) error
	DeleteAttributeValue(ctx context.Context, id uint) error
}

// AttributeQueryRepository จัดการ Read operations
// GetValuesByAttributeID แยกออกมาเป็น method เฉพาะ เพราะ:
//   - บางกรณีต้องการแค่ list of attributes (ไม่ต้อง load values ทุกตัว)
//   - Query Service จะเรียก GetValuesByAttributeID ต่อจาก GetAllAttributes เอง
//     แทนที่จะทำ JOIN ทั้งหมดใน SQL query เดียว (ยืดหยุ่นกว่า)
type AttributeQueryRepository interface {
	GetAllAttributes(ctx context.Context) ([]domain.Attribute, error)
	GetAttributeByID(ctx context.Context, id uint) (*domain.Attribute, error)
	// GetValuesByAttributeID โหลด values ของ attribute ที่ระบุ ใช้คู่กับ GetAllAttributes / GetAttributeByID
	GetValuesByAttributeID(ctx context.Context, attributeID uint) ([]domain.AttributeValue, error)
}
