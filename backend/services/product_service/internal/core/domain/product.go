package domain

import (
	"errors"
	"events"
	"time"
)

type Product struct {
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	Name        string
	Description string
	ImageURLs   []string
	Variants    []ProductVariant
	Categories  []Category
	IsActive    bool // ใช้แยกระหว่าง published (visible) กับ draft/hidden
	CreatedBy   uint // Admin user ID who created the product
	UpdatedBy   uint // Admin user ID who last modified the product
	// domainEvents เก็บ list ของ events ที่เกิดขึ้นใน aggregate นี้
	// WHY เก็บใน slice แทนที่จะ publish ตรงๆ?
	//   - Domain layer ไม่ควรรู้จัก Kafka หรือ Outbox → เก็บไว้ก่อน
	//   - Repo จะดึง events ออกผ่าน PopDomainEvents() แล้วค่อย save ลง outbox table
	//   - ทำให้ Domain Object เป็น pure object ที่ test ได้โดยไม่ต้องมี infrastructure
	domainEvents []events.DomainEvent
}

type ProductVariant struct {
	ID          uint
	ProductID   uint
	Sku         string
	NameVariant string
	Price       float64
	Stock       int
	IsActive    bool
	ImageURLs   []string
	Attributes  []VariantAttribute
}

type ProductFilter struct {
	Page       int
	Limit      int
	Search     string // Filter by product name
	CategoryID uint   // 0 = no category filter
	IsActive   *bool  // nil = all, true = active only, false = inactive only
	SortBy     string
	Order      string
}

// addDomainEvent เพิ่ม event เข้า internal slice (private method)
// WHY ต้องผ่าน method แทนที่จะ append ตรงๆ?
//   - ซ่อน implementation detail ของ event list ไว้ใน aggregate
//   - ถ้าอยากเพิ่ม validation/dedup ทีหลังก็แก้ที่นี่จุดเดียว
func (p *Product) addDomainEvent(event events.DomainEvent) {
	p.domainEvents = append(p.domainEvents, event)
}

// SetActive เปิด/ปิดการแสดงสินค้าต่อ Frontend
// ทำไมแยก method ออกมา?
//   - การ toggle active ไม่ใช่ business rule ที่ซับซ้อน แต่ควรผ่าน Domain method ทุกครั้ง
//   - ทำให้ Service ไม่ต้อง set field โดยตรง (Encapsulation)
func (p *Product) SetActive(active bool, updatedBy uint) {
	p.IsActive = active
	p.UpdatedBy = updatedBy
}

// NewProduct ทำหน้าที่ตรวจสอบความถูกต้อง (Invariants) ก่อนสร้าง Object
func NewProduct(name, description string, imageURLs []string, variants []ProductVariant, categories []Category) (*Product, error) {

	// บังคับ Validate
	if name == "" {
		return nil, errors.New("product name cannot be empty")
	}
	if len(variants) == 0 {
		return nil, errors.New("product must have at least one variant")
	}

	// ถ้าผ่านกฎทั้งหมด ค่อยประกอบร่าง
	return &Product{
		Name:        name,
		Description: description,
		ImageURLs:   imageURLs,
		Variants:    variants,
		Categories:  categories,
	}, nil
}

// SyncIDToEvents injects the DB-assigned product ID into domain events raised before the insert.
// WHY จำเป็น?
//   - ตอน MarkAsCreated() เรียก ID ยังเป็น 0 (DB auto-increment ยังไม่ generate)
//   - หลัง r.GetDB(ctx).Create(productEntity) DB assign ID กลับมา
//   - SyncIDToEvents() ต้องเรียกหลัง Create เพื่อ inject ProductID ที่ถูกต้องลงใน event
func (p *Product) SyncIDToEvents() {
	// variant events ถูก raise เรียงตามลำดับ p.Variants ใน MarkAsCreated
	// → จับคู่ VariantID จริง (ที่ DB assign กลับเข้า p.Variants แล้ว) ตาม index เดียวกัน
	variantIdx := 0
	for _, evt := range p.domainEvents {
		switch e := evt.(type) {
		case *events.ProductCreatedEvent:
			e.ProductID = p.ID
		case *events.ProductVariantAddedEvent:
			e.ProductID = p.ID
			if variantIdx < len(p.Variants) {
				v := p.Variants[variantIdx]
				e.VariantID = v.ID
				// rebuild attributes จาก variant ที่ hydrate Name/Value แล้ว (ตอน raise event ยังมีแค่ ID)
				attrs := make([]events.AttributeKV, len(v.Attributes))
				for i, a := range v.Attributes {
					attrs[i] = events.AttributeKV{Key: a.Name, Value: a.Value}
				}
				e.Attributes = attrs
			}
			variantIdx++
		case *events.ProductImagesUpdatedEvent:
			e.ProductID = p.ID
		case *events.ProductCategoriesUpdatedEvent:
			e.ProductID = p.ID
			cats := make([]events.CategoryKV, len(p.Categories))
			for i, c := range p.Categories {
				cats[i] = events.CategoryKV{CategoryID: c.ID, Name: c.Name, Slug: c.Slug}
			}
			e.Categories = cats
		}
	}
}

// PopDomainEvents ดึง events ทั้งหมดออกมาแล้ว clear slice ให้ว่าง
// WHY ต้อง clear ด้วย?
//   - ป้องกัน double-publish: ถ้าเรียก SaveDomainEvents 2 ครั้ง จะไม่ส่ง event ซ้ำ
//   - Repo เรียก Pop ครั้งเดียวใน SaveDomainEvents แล้วลบออก (one-shot delivery)
func (p *Product) PopDomainEvents() []events.DomainEvent {
	events := p.domainEvents
	p.domainEvents = nil

	return events
}

// MarkAsCreated บันทึกว่าใคร create และ raise ProductCreatedEvent
// WHY raise event ตรงนี้แทนใน Service?
//   - Event เกิดจาก Domain behavior ไม่ใช่ infrastructure concern
//   - Product รู้ว่าตัวเองเพิ่งถูกสร้าง → มันควรประกาศ event เอง
//   - NOTE: ProductID ยังเป็น 0 ณ จุดนี้ → SyncIDToEvents() จะ inject ID จริงหลัง DB insert
func (p *Product) MarkAsCreated(createdBy uint) {
	p.CreatedBy = createdBy
	p.UpdatedBy = createdBy

	// ProductID is 0 at this point; SyncIDToEvents() injects the real ID after DB insert
	p.addDomainEvent(&events.ProductCreatedEvent{
		Name:        p.Name,
		Description: p.Description,
		CreatedBy:   p.CreatedBy,
		OccurredAt:  time.Now(),
	})

	// WHY raise variant/image events ตรงนี้ด้วย?
	//   - ProductCreatedEvent ส่งเฉพาะข้อมูล product ระดับบน (ไม่มี variants/images)
	//   - catalog_service (read model) build document จาก event แยกชนิด → ต้องได้ variant + image event ด้วย
	//   - ถ้าไม่ส่ง สินค้าที่สร้างใหม่จะโผล่ใน catalog แบบไม่มี variant (price = 0) และไม่มีรูป
	//   - AggregateID ของทุก event = ProductID (ดู SaveDomainEvents) → ลำดับใน Kafka partition เดียวกันถูกการันตี
	// NOTE: ProductID/VariantID ยังเป็น 0 ตอนนี้ → SyncIDToEvents() inject ค่าจริงหลัง DB insert
	for _, v := range p.Variants {
		attrs := make([]events.AttributeKV, len(v.Attributes))
		for i, a := range v.Attributes {
			attrs[i] = events.AttributeKV{Key: a.Name, Value: a.Value}
		}
		p.addDomainEvent(&events.ProductVariantAddedEvent{
			Sku:        v.Sku,
			Name:       v.NameVariant,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: attrs,
			OccurredAt: time.Now(),
		})
	}

	if len(p.ImageURLs) > 0 {
		p.addDomainEvent(&events.ProductImagesUpdatedEvent{
			ImageURLs:  p.ImageURLs,
			UpdatedBy:  createdBy,
			OccurredAt: time.Now(),
		})
	}

	// categories ใช้สำหรับ filter ใน catalog → ต้อง sync ด้วย
	// NOTE: ตอนนี้ p.Categories อาจมีแค่ ID (ยังไม่ได้ load Name/Slug)
	//       SyncIDToEvents() จะเติม snapshot จริงหลัง repo โหลด Name/Slug จาก DB
	if len(p.Categories) > 0 {
		p.addDomainEvent(&events.ProductCategoriesUpdatedEvent{
			OccurredAt: time.Now(),
		})
	}
}

// MarkAsDeleted set DeletedAt (soft delete) และ raise ProductDeletedEvent
// WHY Soft Delete?
//   - ไม่ลบจริงจาก DB เพราะอาจต้องการ audit trail หรือ restore ในอนาคต
//   - GORM จะ filter WHERE deleted_at IS NULL ให้อัตโนมัติทุก query
//   - Event ส่งไปบอก downstream (เช่น Cart Service) ว่าสินค้านี้ถูกลบแล้ว
func (p *Product) MarkAsDeleted(deletedBy uint) {
	now := time.Now()
	p.DeletedAt = &now
	p.UpdatedBy = deletedBy

	p.addDomainEvent(&events.ProductDeletedEvent{
		ProductID:  p.ID,
		DeletedBy:  deletedBy,
		OccurredAt: now,
	})
}

// Update General Info
func (p *Product) UpdateInfo(name, description string) error {
	if name == "" {
		return ErrEmptyProductName
	}

	p.Name = name
	p.Description = description
	p.UpdatedAt = time.Now()

	p.addDomainEvent(&events.ProductInfoUpdatedEvent{
		ProductID:   p.ID,
		Name:        name,
		Description: description,
	})

	return nil
}

// RaiseCategoriesUpdated emits a categories-updated event so the read model (catalog)
// can resync the product's category list. The event is enriched with Name/Slug in the
// repository (via SyncIDToEvents) once category rows are loaded.
func (p *Product) RaiseCategoriesUpdated() {
	p.addDomainEvent(&events.ProductCategoriesUpdatedEvent{
		OccurredAt: time.Now(),
	})
}

// UpdateVariantPrice replaces the price on the specified variant and raises a price-changed event.
func (p *Product) UpdateVariantPrice(variantID uint, newPrice float64) error {
	if newPrice < 0 {
		return ErrInvalidInput
	}

	found := false
	for i := range p.Variants {
		if p.Variants[i].ID == variantID {
			variant := &p.Variants[i]

			//oldPrice := p.variants[i].Price // ไม่ควรเข้าถึง p.Variants[i] ตรงๆ เพราะต้องการให้ event มีข้อมูล old price ด้วย
			oldPrice := variant.Price

			//p.Variants[i].Price = newPrice // ไม่ควร set ตรงๆ เพราะต้องการให้ event มีข้อมูล old price ด้วย
			variant.Price = newPrice

			p.addDomainEvent(&events.ProductPriceChangedEvent{
				ProductID:  p.ID,
				VariantID:  variantID,
				OldPrice:   oldPrice,
				NewPrice:   newPrice,
				OccurredAt: time.Now(),
			})

			found = true
			break
		}
	}

	if !found {
		return ErrRecordNotFound
	}

	return nil
}

// UpdateCategories replaces the full category set.
// Callers are responsible for resolving category IDs before invoking this method.
func (p *Product) UpdateCategories(newCategories []Category) {
	p.Categories = newCategories
	p.UpdatedAt = time.Now()
}

// CheckStockAvailability validates and decrements in-memory stock.
// Useful for unit testing; production paths use Repo.DecreaseStock for atomic DB-level enforcement.
//
// WHY แยกเป็น 2 path:
//   - Unit Test: ใช้ method นี้เพราะไม่มี DB (test ใน memory ล้วนๆ)
//   - Production: ใช้ Repo.DecreaseStock เพราะต้องการ Atomic SQL UPDATE
//     เพื่อป้องกัน Race Condition (concurrent orders ซื้อของชิ้นสุดท้ายพร้อมกัน)
func (p *Product) CheckStockAvailability(variantID uint, qty int) error {
	for i, v := range p.Variants {
		if v.ID == variantID {
			if v.Stock < qty {
				return ErrInvalidInput
			}

			p.Variants[i].Stock -= qty
			return nil
		}
	}
	return ErrRecordNotFound
}

// Add New Variant
func (p *Product) AddNewVariant(v ProductVariant) {
	p.Variants = append(p.Variants, v)

	attrs := make([]events.AttributeKV, len(v.Attributes))
	for i, a := range v.Attributes {
		attrs[i] = events.AttributeKV{Key: a.Name, Value: a.Value}
	}

	// v.ID must be populated from the DB before calling this method
	p.addDomainEvent(&events.ProductVariantAddedEvent{
		ProductID:  p.ID,
		VariantID:  v.ID,
		Sku:        v.Sku,
		Name:       v.NameVariant,
		Price:      v.Price,
		Stock:      v.Stock,
		Attributes: attrs,
		OccurredAt: time.Now(),
	})
}

// UpdateProductImages replaces the product-level image list.
func (p *Product) UpdateProductImages(imageURLs []string, updatedBy uint) {
	p.ImageURLs = imageURLs
	p.UpdatedBy = updatedBy
	p.UpdatedAt = time.Now()

	p.addDomainEvent(&events.ProductImagesUpdatedEvent{
		ProductID:  p.ID,
		ImageURLs:  imageURLs,
		UpdatedBy:  updatedBy,
		OccurredAt: p.UpdatedAt,
	})
}

// UpdateVariantImages replaces image list of a specific variant.
func (p *Product) UpdateVariantImages(variantID uint, imageURLs []string, updatedBy uint) error {
	for i := range p.Variants {
		if p.Variants[i].ID == variantID {
			p.Variants[i].ImageURLs = imageURLs
			p.UpdatedBy = updatedBy
			p.UpdatedAt = time.Now()

			p.addDomainEvent(&events.ProductVariantImagesUpdatedEvent{
				ProductID:  p.ID,
				VariantID:  variantID,
				ImageURLs:  imageURLs,
				UpdatedBy:  updatedBy,
				OccurredAt: p.UpdatedAt,
			})
			return nil
		}
	}
	return ErrRecordNotFound
}

// Adjust Stock for Stock Take or Damage
func (p *Product) AdjustStock(variantID uint, newStock int, reason string, adjustedBy uint) error {
	for i := range p.Variants {
		if p.Variants[i].ID == variantID {
			oldStock := p.Variants[i].Stock

			p.Variants[i].Stock = newStock
			p.UpdatedBy = adjustedBy
			p.UpdatedAt = time.Now()

			p.addDomainEvent(&events.StockAdjustedEvent{
				ProductID:  p.ID,
				VariantID:  variantID,
				OldStock:   oldStock,
				NewStock:   newStock,
				Reason:     reason,
				AdjustedBy: adjustedBy,
				OccurredAt: time.Now(),
			})
			return nil
		}
	}
	return ErrRecordNotFound
}
