// WHAT: Port interface สำหรับ Catalog Service client
// WHY แยก interface ออกมา?
//   - Hexagonal Architecture: service layer depend on abstraction ไม่ใช่ concrete HTTP client
//   - Testable: mock ได้ง่ายใน unit test ไม่ต้องมี real catalog_service
//   - Swappable: เปลี่ยนจาก REST เป็น gRPC โดยไม่กระทบ service layer
package gateway

import "context"

// VariantSnapshot คือข้อมูลที่ order_service ต้องการจาก catalog_service
// WHY ไม่ embed struct จาก catalog_service?
//   - Anti-Corruption Layer: ป้องกัน coupling ระหว่าง service boundaries
//   - order_service เลือก field ที่ต้องการเอง
type VariantSnapshot struct {
	VariantID   uint
	ProductName string
	VariantName string
	Price       float64
	ImageURL    string
}

// CatalogClient interface สำหรับ fetch ข้อมูล variant จาก catalog_service
// ใช้ใน PlaceOrder เพื่อ:
//   1. server-side price lookup (ป้องกัน price tampering)
//   2. denormalize product_name, variant_name, image_url ลง order_items
type CatalogClient interface {
	GetVariantSnapshot(ctx context.Context, variantID uint) (*VariantSnapshot, error)
}
