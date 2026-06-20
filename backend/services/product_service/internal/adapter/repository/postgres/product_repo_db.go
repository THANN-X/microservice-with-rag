package repository

import (
	"context"
	"database"
	"encoding/json"
	"errors"
	"fmt"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"
	port "product_service/internal/core/port/repo"

	"gorm.io/gorm"
)

type productRepository struct {
	*database.TxHelper
}

func NewProductRepository(db *gorm.DB) (port.ProductCommandRepository, port.ProductQueryRepository) {
	repo := &productRepository{
		TxHelper: database.NewTxHelper(db),
	}
	return repo, repo
}

// getAllSubCategoryIDs ใช้ BFS (Breadth-First Search) ดึง ID ของ category และลูกหลานทั้งหมด
// WHY BFS แทน recursive SQL (CTE)?
//   - CTE recursive ดีกว่าสำหรับ tree ลึกมาก แต่โค้ดยากกว่า
//   - BFS ใน Go อ่านง่ายกว่า และ e-commerce ปกติพนักงานไม่เกิน 3-4 ระดับ
//   - Trade-off: N queries สำหรับ N ระดับชั้น (Suitable for ~4 levels deep)
//
// TODO: ถ้า category tree ลึกมาก ให้พิจารณา Materialized Path หรือ Closure Table
func (r *productRepository) getAllSubCategoryIDs(ctx context.Context, rootID uint) ([]uint, error) {
	allIDs := []uint{rootID}
	currentLevelIDs := []uint{rootID}

	for len(currentLevelIDs) > 0 {
		var childrenIDs []uint

		if err := r.GetDB(ctx).Model(&entity.CategoryEntity{}).
			Where("parent_id IN ?", currentLevelIDs).
			Pluck("id", &childrenIDs).Error; err != nil {
			return nil, err
		}

		if len(childrenIDs) > 0 {
			allIDs = append(allIDs, childrenIDs...)
		}
		currentLevelIDs = childrenIDs
	}

	return allIDs, nil
}

// COMMAND IMPLEMENTATION
// ==========================================================
// WHY ใช้ struct เดียว implement ทั้ง Command &amp; Query Repository?
//   - Product service ยังโตไม่ถึงจุดที่ต้อง read replica แยก
//   - แต่ interface แยกไว้แล้ว ถ้าวันหนึ่งต้องการแยก ก็แค่ implement QueryRepository ใหม่โดยไม่กระทบ Command
func (r *productRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	productEntity := entity.ToProductEntity(product)

	if err := r.GetDB(ctx).Create(productEntity).Error; err != nil {
		return err
	}

	// Sync DB-generated fields กลับไปยัง domain object
	// WHY? Service layer อาจต้องการ ID ใน step ถัดไป (e.g. SyncIDToEvents)
	product.ID = productEntity.ID
	product.CreatedAt = productEntity.CreatedAt
	product.UpdatedAt = productEntity.UpdatedAt

	// sync DB-assigned variant IDs กลับเข้า domain (GORM create variants ตามลำดับ slice เดิม)
	// → SyncIDToEvents() จะ inject VariantID ที่ถูกต้องลงใน ProductVariantAddedEvent ได้
	for i := range product.Variants {
		if i < len(productEntity.Variants) {
			product.Variants[i].ID = productEntity.Variants[i].ID
		}
	}

	// โหลด Name/Slug ของ categories (ตอน create มีแค่ ID) เพื่อให้ ProductCategoriesUpdatedEvent
	// ส่ง snapshot ครบ → catalog embed แล้ว filter ตาม category ได้
	r.loadCategoryNames(ctx, product)

	// โหลด Name/Value ของ variant attributes (ตอน create มีแค่ AttributeValue ID) เพื่อให้
	// ProductVariantAddedEvent ส่ง attribute snapshot ครบ → catalog แสดงปุ่มตัวเลือก variant ได้
	ptrs := make([]*domain.ProductVariant, len(product.Variants))
	for i := range product.Variants {
		ptrs[i] = &product.Variants[i]
	}
	r.hydrateVariantAttributes(ctx, ptrs)

	product.SyncIDToEvents() // inject real ID เข้าไปใน domain events ที่สร้างไว้ก่อน insert

	return r.SaveDomainEvents(ctx, product)
}

// loadCategoryNames hydrates Name/Slug on product.Categories (which may carry only IDs)
// so that ProductCategoriesUpdatedEvent can ship a complete snapshot to the catalog read model.
func (r *productRepository) loadCategoryNames(ctx context.Context, product *domain.Product) {
	if len(product.Categories) == 0 {
		return
	}
	ids := make([]uint, len(product.Categories))
	for i, c := range product.Categories {
		ids[i] = c.ID
	}
	var catEntities []entity.CategoryEntity
	if err := r.GetDB(ctx).Where("id IN ?", ids).Find(&catEntities).Error; err != nil {
		return
	}
	byID := make(map[uint]entity.CategoryEntity, len(catEntities))
	for _, c := range catEntities {
		byID[c.ID] = c
	}
	for i := range product.Categories {
		if c, ok := byID[product.Categories[i].ID]; ok {
			product.Categories[i].Name = c.Name
			product.Categories[i].Slug = c.Slug
		}
	}
}

// hydrateVariantAttributes เติม Name/Value ลงใน variant.Attributes ที่มีแค่ AttributeValue ID
// WHY ต้องมี? — ตอนสร้าง/เพิ่ม variant domain object มีแค่ ID (ใช้ผูก many2many join table)
//
//	แต่ ProductVariantAddedEvent ต้องส่ง Name/Value จริงไปให้ catalog read model
//	ถ้าไม่ hydrate → catalog เก็บ attribute เป็น {key:"", value:""} → ปุ่มตัวเลือก variant ว่างเปล่า
func (r *productRepository) hydrateVariantAttributes(ctx context.Context, variants []*domain.ProductVariant) {
	// รวบรวม AttributeValue ID ทั้งหมดจากทุก variant (dedupe)
	idSet := make(map[uint]struct{})
	for _, v := range variants {
		for _, a := range v.Attributes {
			if a.ID != 0 {
				idSet[a.ID] = struct{}{}
			}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	var avEntities []entity.AttributeValueEntity
	if err := r.GetDB(ctx).Preload("Attribute").Where("id IN ?", ids).Find(&avEntities).Error; err != nil {
		return
	}
	byID := make(map[uint]entity.AttributeValueEntity, len(avEntities))
	for _, av := range avEntities {
		byID[av.ID] = av
	}

	for _, v := range variants {
		for j := range v.Attributes {
			if av, ok := byID[v.Attributes[j].ID]; ok {
				v.Attributes[j].Value = av.Value
				v.Attributes[j].Name = av.Attribute.Name
			}
		}
	}
}

func (r *productRepository) GetProductByID(ctx context.Context, id uint) (*domain.Product, error) {
	productEntity := &entity.ProductEntity{}

	// Preload หลายระดับเพื่อ Load complete Aggregate:
	// Variants → AttributeValues → Attribute (Name)
	// WHY ต้อง Preload ลึก 3 ชั้น?
	//   - Handler ต้องการ Attribute Name/Value สำหรับ response
	//   - Preload เข้า GORM (ไม่ใช่ JOIN) → query แยกต่างหาก สะอาดกว่า nested JOIN สำหรับโครงสร้างนี้
	err := r.GetDB(ctx).Preload("Variants").Preload("Variants.AttributeValues").Preload("Variants.AttributeValues.Attribute").Preload("Categories").First(productEntity, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRecordNotFound // หรือ return custom error
		}
		return nil, err
	}

	return productEntity.ToProductDomain(), nil
}

func (r *productRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {

	productEntity := entity.ToProductEntity(product)

	// Omit associations เพราะ Save แบบ eager จะ upsert Variant/Category ทั้งก้อน
	// ซึ่งอาจลบ Variants ที่ไม่ได้ส่งมา → อันตราย ทำ Associations.Replace แทน
	result := r.GetDB(ctx).Omit("Variants", "Categories").Save(productEntity)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNoDataModified
	}
	// Categories ใช้ many2many (join table: product_categories) → Association.Replace ถูกต้อง
	// Replace จะลบ row ใน join table ทั้งหมดแล้ว insert ชุดใหม่ → ไม่กระทบ category row จริงๆ
	if product.Categories != nil {
		if err := r.GetDB(ctx).Model(productEntity).Association("Categories").Replace(productEntity.Categories); err != nil {
			return err
		}
	}

	if product.Variants != nil {
		// Association.Replace manages FK only — it does NOT update non-FK fields (e.g. price, stock)
		// on existing rows. Use Save per variant to issue a full UPDATE for each row.
		for i := range productEntity.Variants {
			if err := r.GetDB(ctx).Save(&productEntity.Variants[i]).Error; err != nil {
				return err
			}
		}
	}

	product.UpdatedAt = productEntity.UpdatedAt

	// categories อาจเปลี่ยนตอน update → hydrate Name/Slug แล้ว inject ลง ProductCategoriesUpdatedEvent
	// (ถ้า service ได้ raise event ไว้) เพื่อให้ catalog resync ได้ครบ
	r.loadCategoryNames(ctx, product)
	product.SyncIDToEvents()

	return r.SaveDomainEvents(ctx, product)
}

func (r *productRepository) DeleteProduct(ctx context.Context, product *domain.Product) error {
	// Soft delete ทุก Variant ของ product นี้ก่อน
	// WHY ต้องทำแยก?
	//   - GORM soft delete (UpdateColumn deleted_at) ไม่ได้ cascade ลงไปที่ child rows อัตโนมัติ
	//   - constraint OnDelete:CASCADE จะทำงานก็ต่อเมื่อเป็น hard delete เท่านั้น
	//   - soft delete ต้องจัดการ cascade เองใน application layer
	if err := r.GetDB(ctx).Where("product_id = ?", product.ID).Delete(&entity.ProductVariantEntity{}).Error; err != nil {
		return err
	}

	result := r.GetDB(ctx).Delete(&entity.ProductEntity{}, product.ID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return r.SaveDomainEvents(ctx, product)
}

func (r *productRepository) AddVariant(ctx context.Context, variant *domain.ProductVariant) error {
	vEntity := entity.ToProductVariantEntity(variant)

	if err := r.GetDB(ctx).Create(vEntity).Error; err != nil {
		return err
	}

	// Sync ID กลับไป Domain → Service จะส่งต่อ ID นี้ให้ AddNewVariant() raise event
	variant.ID = vEntity.ID

	// เติม Name/Value ของ attributes (variant ที่ส่งมามีแค่ AttributeValue ID)
	// เพื่อให้ ProductVariantAddedEvent ส่ง attribute snapshot ครบให้ catalog
	r.hydrateVariantAttributes(ctx, []*domain.ProductVariant{variant})

	return nil
}

func (r *productRepository) UpdateStock(ctx context.Context, variantID uint, newStock int) error {
	// ใช้ targeted UPDATE แทน Save เพื่อ update เฉพาะ field stock ไม่ยุ่งกับ field อื่น
	result := r.GetDB(ctx).Model(&entity.ProductVariantEntity{}).
		Where("id = ?", variantID).
		Update("stock", newStock)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

// SetProductActive ทำ targeted UPDATE is_active=? แทน full Save
// เหตุผล: หลีก unnecessary field updates และ race condition กับ concurrent editors
func (r *productRepository) SetProductActive(ctx context.Context, productID uint, active bool) error {
	result := r.GetDB(ctx).Model(&entity.ProductEntity{}).
		Where("id = ?", productID).
		Update("is_active", active)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrRecordNotFound
	}
	return nil
}

// SetVariantActive ทำ targeted UPDATE is_active=? สำหรับ Variant อย่างเดียว
func (r *productRepository) SetVariantActive(ctx context.Context, variantID uint, active bool) error {
	result := r.GetDB(ctx).Model(&entity.ProductVariantEntity{}).
		Where("id = ?", variantID).
		Update("is_active", active)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrRecordNotFound
	}

	return nil
}

func (r *productRepository) DecreaseStock(ctx context.Context, variantID uint, qty int) error {
	productVariantEntity := entity.ProductVariantEntity{}
	// Atomic conditional UPDATE: stock = stock - qty WHERE stock >= qty
	// WHY ใช้ SQL condition แทนเช็คใน Go?
	//   - ถ้าเช็คใน Go: Load stock → check → update มี race condition
	//     (2 requests อื่น load stock=1 พร้อมกัน ส่ง qty=1 ทั้งคู่ → stock ติดลบเป็น -1)
	//   - SQL atomic update ป้องกัน race condition โดยอัตโนมัติ
	//   - ถ้า RowsAffected=0 → หมายความว่า stock ไม่พอ → รับ ErrNoDataModified → พ่น InsufficientStockError
	result := r.GetDB(ctx).Model(productVariantEntity).Where("id = ? AND stock >= ?", variantID, qty).Update("stock", gorm.Expr("stock - ?", qty))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNoDataModified
	}

	return nil
}

func (r *productRepository) IncreaseStock(ctx context.Context, variantID uint, qty int) error {
	productVariantEntity := entity.ProductVariantEntity{}
	// Atomic UPDATE stock = stock + qty (ไม่มี condition เพราะการคืน stock ไม่มีขีดจำกัดด้านบน)
	result := r.GetDB(ctx).Model(productVariantEntity).Where("id = ?", variantID).Update("stock", gorm.Expr("stock + ?", qty))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNoDataModified
	}

	return nil
}

// GetVariantStockInfo อ่าน product_id + stock ปัจจุบันของ variant — ใช้หลังตัด/คืน stock
// เพื่อ emit StockUpdatedEvent แบบ absolute (catalog เอาไป set ทับ)
func (r *productRepository) GetVariantStockInfo(ctx context.Context, variantID uint) (uint, int, error) {
	var v entity.ProductVariantEntity
	if err := r.GetDB(ctx).Select("product_id", "stock").Where("id = ?", variantID).First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, domain.ErrRecordNotFound
		}
		return 0, 0, err
	}
	return v.ProductID, v.Stock, nil
}

func (r *productRepository) SaveDomainEvents(ctx context.Context, agg *domain.Product) error {
	// PopDomainEvents ดึงและ clear events พร้อมกันในครั้งเดียว → ป้องกัน double-publish
	events := agg.PopDomainEvents()

	if len(events) == 0 {
		return nil
	}

	for _, evt := range events {

		payloadBytes, err := json.Marshal(evt)

		if err != nil {
			return err
		}

		outboxEvent := domain.NewOutboxMessage(
			"product.events",
			fmt.Sprintf("%d", agg.ID),
			"PRODUCT",
			evt.EventName(),
			string(payloadBytes),
		)

		outboxEntity := entity.ToOutboxEventEntity(outboxEvent)

		if err := r.GetDB(ctx).Create(outboxEntity).Error; err != nil {
			return err
		}
	}
	return nil

}

// QUERY IMPLEMENTATION
func (r *productRepository) FindByID(ctx context.Context, id uint) (*domain.Product, error) {
	productEntity := &entity.ProductEntity{}

	err := r.GetDB(ctx).Preload("Variants").Preload("Variants.AttributeValues").Preload("Variants.AttributeValues.Attribute").Preload("Categories").First(productEntity, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRecordNotFound // หรือ return custom error
		}
		return nil, err
	}

	return productEntity.ToProductDomain(), nil
}

func (r *productRepository) FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	var entities []entity.ProductEntity
	var total int64

	query := r.GetDB(ctx).Model(&entity.ProductEntity{})

	if filter.Search != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Search+"%")
	}

	if filter.CategoryID > 0 {
		// Include the root category and all its descendants
		targetCategoryIDs, err := r.getAllSubCategoryIDs(ctx, filter.CategoryID)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("id IN (?)",
			r.GetDB(ctx).Table("product_categories").
				Select("product_id").
				Where("category_id IN ?", targetCategoryIDs),
		)
	}

	// กรอง is_active: nil = ดึงทั้งหมด, true = active, false = inactive
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Whitelist sort fields and order direction to prevent SQL injection
	// WHY whitelist แทน parameterized query?
	//   - ORDER BY clause ใน SQL ไม่รองรับ placeholder ($1) เหมือน WHERE clause
	//   - ถ้าส่ง filter.SortBy = "id; DROP TABLE products--" จะเป็น SQL injection
	//   - whitelist map แก้ปัญหานี้ได้อย่างหรัด (constant time lookup)
	allowedSortFields := map[string]bool{"created_at": true, "updated_at": true, "name": true, "price": true}
	allowedOrders := map[string]bool{"asc": true, "desc": true}
	if !allowedSortFields[filter.SortBy] {
		filter.SortBy = "created_at"
	}
	if !allowedOrders[filter.Order] {
		filter.Order = "desc"
	}

	offset := (filter.Page - 1) * filter.Limit
	orderClause := fmt.Sprintf("%s %s", filter.SortBy, filter.Order)

	err := query.Preload("Variants").Preload("Variants.AttributeValues").Preload("Variants.AttributeValues.Attribute").Preload("Categories").
		Limit(filter.Limit).
		Offset(offset).
		Order(orderClause).
		Find(&entities).Error

	if err != nil {
		return nil, 0, err
	}

	// Map to Domain
	results := make([]domain.Product, len(entities))

	for i, e := range entities {
		results[i] = *e.ToProductDomain()
	}

	return results, total, nil
}
