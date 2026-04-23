package domain

// VariantAttribute เป็น read-only projection ที่ใช้ตอน load Product Variant ขึ้นมาแสดงผล
// ทำไมถึงแยกออกมาจาก Attribute ปกติ?
//   - ตอน Query Product เราไม่ต้องการ ID ของ Attribute ต้นทาง แค่ต้องการ Name+Value
//   - Struct แบบนี้ทำให้ Mapper จาก DB Entity → Domain ง่ายขึ้น ไม่ต้องพา field ที่ไม่จำเป็นมาด้วย
type VariantAttribute struct {
	ID    uint
	Name  string // e.g. "Color"
	Value string // e.g. "Red"
}

// Attribute คือ Domain Object สำหรับ Admin จัดการประเภทของ Attribute (e.g. "Color", "Size")
// แยกออกมาจาก VariantAttribute เพราะเป็นคนละ Use Case:
//   - Attribute ใช้ใน Admin CRUD (Create/Edit/Delete ชื่อ Attribute)
//   - VariantAttribute ใช้ใน Product Query (แสดงผล Option ของ Variant)
type Attribute struct {
	ID   uint
	Name string
}

// Update เป็น domain method สำหรับแก้ไข Attribute
// บังคับให้ทุก update ผ่าน method นี้เสมอ เพื่อให้ business rules ถูก enforce ที่ Domain layer
// (ตาม pattern เดียวกับ Category.UpdateCategory และ Product.UpdateInfo)
func (a *Attribute) Update(name string) {
	a.Name = name
}

// AttributeValue คือค่าที่เป็นไปได้ของ Attribute แต่ละตัว
// เช่น Attribute "Color" จะมี Values: "Red", "Blue", "Green"
// เหตุที่เก็บ AttributeID ไว้ใน Domain เพื่อให้ Service Layer รู้ว่า Value นี้อยู่ภายใต้ Attribute ไหน
// ก่อน Insert โดยไม่ต้องโหลด Attribute ทั้งก้อนมา
type AttributeValue struct {
	ID          uint
	AttributeID uint // FK ชี้ไปยัง Attribute ต้นทาง
	Value       string
}
