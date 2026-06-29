package mongo

import (
	"catalog_service/internal/core/domain"
	repo "catalog_service/internal/core/port/repo"
	"context"
	"errors"
	"fmt"
	"time"

	"errs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"logs"
)

const catalogCollection = "catalog_products"

// catalogDocument คือ MongoDB BSON document สำหรับ catalog_products collection
type catalogDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ProductID   uint               `bson:"product_id"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	ImageURLs   []string           `bson:"image_urls"`
	Categories  []categoryDoc      `bson:"categories"`
	Variants    []variantDoc       `bson:"variants"`
	IsActive    bool               `bson:"is_active"`
	IsDeleted   bool               `bson:"is_deleted"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

type variantDoc struct {
	VariantID  uint           `bson:"variant_id"`
	Sku        string         `bson:"sku"`
	Name       string         `bson:"name"`
	Price      float64        `bson:"price"`
	Stock      int            `bson:"stock"`
	IsActive   bool           `bson:"is_active"`
	ImageURLs  []string       `bson:"image_urls"`
	Attributes []attributeDoc `bson:"attributes"`
	// UpdatedAt = OccurredAt ของ event ล่าสุดที่แก้ variant นี้ — ใช้เป็น Timestamp Guard
	// กัน event เก่าที่หลงมาทีหลัง (out-of-order) มาทับค่าที่ใหม่กว่า
	UpdatedAt time.Time `bson:"updated_at,omitempty"`
}

type categoryDoc struct {
	CategoryID uint   `bson:"category_id"`
	Name       string `bson:"name"`
	Slug       string `bson:"slug"`
}

type attributeDoc struct {
	Key   string `bson:"key"`
	Value string `bson:"value"`
}

type catalogRepository struct {
	col *mongo.Collection
}

// NewCatalogRepository คืน write และ read repo จาก struct เดียวกัน
func NewCatalogRepository(db *mongo.Database) (repo.CatalogWriteRepository, repo.CatalogReadRepository) {
	r := &catalogRepository{col: db.Collection(catalogCollection)}
	return r, r
}

// EnsureIndexes สร้าง indexes ที่จำเป็นหากยังไม่มี (idempotent)
func EnsureIndexes(db *mongo.Database) error {
	col := db.Collection(catalogCollection)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			// unique index สำหรับ product_id — ใช้เป็น lookup key หลัก
			Keys:    bson.D{{Key: "product_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			// text index สำหรับ full-text search ทั้ง name และ description
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "description", Value: "text"},
			},
		},
		{
			// index สำหรับ filter by category
			Keys: bson.D{{Key: "categories.category_id", Value: 1}},
		},
		{
			// compound index สำหรับ filter active, non-deleted products (query หลักของ listing)
			Keys: bson.D{
				{Key: "is_active", Value: 1},
				{Key: "is_deleted", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

// --- Write Methods ---

func (r *catalogRepository) Upsert(ctx context.Context, product *domain.CatalogProduct) error {
	now := time.Now()
	categories := toCategoryDocs(product.Categories)
	variants := toVariantDocs(product.Variants)

	filter := bson.M{"product_id": product.ProductID}
	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"description": product.Description,
			"image_urls":  product.ImageURLs,
			"categories":  categories,
			"variants":    variants,
			"is_active":   product.IsActive,
			"is_deleted":  product.IsDeleted,
			"updated_at":  now,
		},
		// created_at ตั้งแค่ครั้งแรก ไม่เขียนทับ
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	_, err := r.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *catalogRepository) UpdateInfo(ctx context.Context, productID uint, name, description string) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"name":        name,
			"description": description,
			"updated_at":  time.Now(),
		},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("product not found in catalog")
	}
	return nil
}

func (r *catalogRepository) UpdateVariantPrice(ctx context.Context, productID uint, variantID uint, newPrice float64, occurredAt time.Time) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"variants.$[elem].price":      newPrice,
			"variants.$[elem].updated_at": occurredAt,
			"updated_at":                  time.Now(),
		},
	}
	// Timestamp Guard: แก้เฉพาะ variant ที่ updated_at เก่ากว่า event (หรือยังไม่เคยมี) → ปัด event เก่าที่มาช้าทิ้ง
	opts := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{bson.M{
			"elem.variant_id": variantID,
			"$or": []bson.M{
				{"elem.updated_at": bson.M{"$exists": false}},
				{"elem.updated_at": bson.M{"$lt": occurredAt}},
			},
		}},
	})

	result, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		// ไม่เจอ product document — event อาจมาก่อน PRODUCT_CREATED (gap) → log ไว้ตรวจสอบ ไม่ block partition
		logs.Warn(fmt.Sprintf("catalog: UpdateVariantPrice no matching product (gap?) product_id=%d variant_id=%d", productID, variantID))
	}
	return nil
}

func (r *catalogRepository) AddVariant(ctx context.Context, productID uint, variant domain.EmbeddedVariant) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$push": bson.M{"variants": toVariantDoc(variant)},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("product not found in catalog")
	}
	return nil
}

func (r *catalogRepository) UpdateVariantStock(ctx context.Context, productID uint, variantID uint, newStock int, occurredAt time.Time) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"variants.$[elem].stock":      newStock,
			"variants.$[elem].updated_at": occurredAt,
			"updated_at":                  time.Now(),
		},
	}
	// Timestamp Guard: แก้เฉพาะ variant ที่ updated_at เก่ากว่า event (หรือยังไม่เคยมี) → ปัด event เก่าที่มาช้าทิ้ง
	opts := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{bson.M{
			"elem.variant_id": variantID,
			"$or": []bson.M{
				{"elem.updated_at": bson.M{"$exists": false}},
				{"elem.updated_at": bson.M{"$lt": occurredAt}},
			},
		}},
	})

	result, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		// ไม่เจอ product document — event อาจมาก่อน PRODUCT_CREATED (gap) → log ไว้ตรวจสอบ ไม่ block partition
		logs.Warn(fmt.Sprintf("catalog: UpdateVariantStock no matching product (gap?) product_id=%d variant_id=%d", productID, variantID))
	}
	return nil
}

func (r *catalogRepository) MarkDeleted(ctx context.Context, productID uint) error {
	filter := bson.M{"product_id": productID}
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"is_active":  false,
			"updated_at": time.Now(),
		},
	}

	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *catalogRepository) UpdateProductImages(ctx context.Context, productID uint, imageURLs []string) error {
	if imageURLs == nil {
		imageURLs = []string{}
	}
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"image_urls": imageURLs,
			"updated_at": time.Now(),
		},
	}
	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("product not found in catalog")
	}
	return nil
}

func (r *catalogRepository) UpdateVariantImages(ctx context.Context, productID uint, variantID uint, imageURLs []string) error {
	if imageURLs == nil {
		imageURLs = []string{}
	}
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"variants.$[elem].image_urls": imageURLs,
			"updated_at":                  time.Now(),
		},
	}
	opts := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{bson.M{"elem.variant_id": variantID}},
	})
	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	return err
}

// UpdateCategories แทนที่ category list ทั้งใบ (idempotent — set ทับทั้งหมด)
func (r *catalogRepository) UpdateCategories(ctx context.Context, productID uint, categories []domain.EmbeddedCategory) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"categories": toCategoryDocs(categories),
			"updated_at": time.Now(),
		},
	}
	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("product not found in catalog")
	}
	return nil
}

// SetActive ตั้ง is_active ระดับ product (ซ่อน/แสดงสินค้าบนหน้าเว็บ)
// WHY filter is_deleted:false? — สินค้าที่ลบไปแล้วไม่ควรถูกปลุกกลับมาด้วยการ toggle active
func (r *catalogRepository) SetActive(ctx context.Context, productID uint, active bool) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"is_active":  active,
			"updated_at": time.Now(),
		},
	}
	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("product not found in catalog")
	}
	return nil
}

// SetVariantActive ตั้ง is_active ระดับ variant (ใช้ arrayFilters)
func (r *catalogRepository) SetVariantActive(ctx context.Context, productID uint, variantID uint, active bool, occurredAt time.Time) error {
	filter := bson.M{"product_id": productID, "is_deleted": false}
	update := bson.M{
		"$set": bson.M{
			"variants.$[elem].is_active":  active,
			"variants.$[elem].updated_at": occurredAt,
			"updated_at":                  time.Now(),
		},
	}
	// Timestamp Guard: แก้เฉพาะ variant ที่ updated_at เก่ากว่า event (หรือยังไม่เคยมี) → ปัด event เก่าที่มาช้าทิ้ง
	opts := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{bson.M{
			"elem.variant_id": variantID,
			"$or": []bson.M{
				{"elem.updated_at": bson.M{"$exists": false}},
				{"elem.updated_at": bson.M{"$lt": occurredAt}},
			},
		}},
	})
	result, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		// ไม่เจอ product document — event อาจมาก่อน PRODUCT_CREATED (gap) → log ไว้ตรวจสอบ ไม่ block partition
		logs.Warn(fmt.Sprintf("catalog: SetVariantActive no matching product (gap?) product_id=%d variant_id=%d", productID, variantID))
	}
	return nil
}

func (r *catalogRepository) FindByProductID(ctx context.Context, productID uint) (*domain.CatalogProduct, error) {
	filter := bson.M{"product_id": productID, "is_deleted": false}

	var doc catalogDocument
	if err := r.col.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errs.NewNotFoundError("product not found")
		}
		return nil, err
	}

	return toDomain(&doc), nil
}

func (r *catalogRepository) FindByVariantID(ctx context.Context, variantID uint) (*domain.CatalogProduct, *domain.EmbeddedVariant, error) {
	filter := bson.M{
		"is_deleted": false,
		"variants":   bson.M{"$elemMatch": bson.M{"variant_id": variantID}},
	}

	var doc catalogDocument
	if err := r.col.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, errs.NewNotFoundError("variant not found")
		}
		return nil, nil, err
	}

	product := toDomain(&doc)
	for i := range product.Variants {
		if product.Variants[i].VariantID == variantID {
			v := product.Variants[i]
			return product, &v, nil
		}
	}

	return nil, nil, errs.NewNotFoundError("variant not found")
}

func (r *catalogRepository) FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.CatalogProduct, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}

	query := bson.M{"is_deleted": false, "is_active": true}

	if len(filter.CategoryIDs) > 0 {
		query["categories"] = bson.M{"$elemMatch": bson.M{"category_id": bson.M{"$in": filter.CategoryIDs}}}
	} else if filter.CategoryID > 0 {
		query["categories"] = bson.M{"$elemMatch": bson.M{"category_id": filter.CategoryID}}
	}
	if filter.Search != "" {
		query["$text"] = bson.M{"$search": filter.Search}
	}

	sortField := filter.SortBy
	if sortField == "" {
		sortField = "created_at"
	}
	sortOrder := -1
	if filter.Order == "asc" {
		sortOrder = 1
	}

	total, err := r.col.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := r.col.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var docs []catalogDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, err
	}

	products := make([]domain.CatalogProduct, len(docs))
	for i, d := range docs {
		products[i] = *toDomain(&d)
	}

	return products, total, nil
}

// --- Internal converters ---

func toCategoryDocs(cats []domain.EmbeddedCategory) []categoryDoc {
	docs := make([]categoryDoc, len(cats))
	for i, c := range cats {
		docs[i] = categoryDoc{CategoryID: c.CategoryID, Name: c.Name, Slug: c.Slug}
	}
	return docs
}

func toVariantDoc(v domain.EmbeddedVariant) variantDoc {
	attrs := make([]attributeDoc, len(v.Attributes))
	for i, a := range v.Attributes {
		attrs[i] = attributeDoc{Key: a.Key, Value: a.Value}
	}
	return variantDoc{
		VariantID:  v.VariantID,
		Sku:        v.Sku,
		Name:       v.Name,
		Price:      v.Price,
		Stock:      v.Stock,
		IsActive:   v.IsActive,
		ImageURLs:  v.ImageURLs,
		Attributes: attrs,
	}
}

func toVariantDocs(variants []domain.EmbeddedVariant) []variantDoc {
	docs := make([]variantDoc, len(variants))
	for i, v := range variants {
		docs[i] = toVariantDoc(v)
	}
	return docs
}

func toDomain(doc *catalogDocument) *domain.CatalogProduct {
	categories := make([]domain.EmbeddedCategory, len(doc.Categories))
	for i, c := range doc.Categories {
		categories[i] = domain.EmbeddedCategory{CategoryID: c.CategoryID, Name: c.Name, Slug: c.Slug}
	}

	variants := make([]domain.EmbeddedVariant, len(doc.Variants))
	for i, v := range doc.Variants {
		attrs := make([]domain.VariantAttribute, len(v.Attributes))
		for j, a := range v.Attributes {
			attrs[j] = domain.VariantAttribute{Key: a.Key, Value: a.Value}
		}
		variants[i] = domain.EmbeddedVariant{
			VariantID:  v.VariantID,
			Sku:        v.Sku,
			Name:       v.Name,
			Price:      v.Price,
			Stock:      v.Stock,
			IsActive:   v.IsActive,
			ImageURLs:  v.ImageURLs,
			Attributes: attrs,
		}
	}

	return &domain.CatalogProduct{
		ID:          doc.ID.Hex(),
		ProductID:   doc.ProductID,
		Name:        doc.Name,
		Description: doc.Description,
		ImageURLs:   doc.ImageURLs,
		Categories:  categories,
		Variants:    variants,
		IsActive:    doc.IsActive,
		IsDeleted:   doc.IsDeleted,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}
