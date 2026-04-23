package mapper

import (
	"product_service/internal/core/domain"
	dto "product_service/internal/core/port/service/dto"
)

// ToDomain_CreateProduct แปลง CreateProductReq เป็น incomplete Domain Object
// NOTE: function นี้ไม่ถูกใช้ใน productCommandService ในตอนนี้
// Service ใช้ ToDomain_Variants/ToDomain_Categories แล้วส่ง domain.NewProduct() แทน
// TODO: refactor ตัด function นี้ออก หรือ migrate มาใช้ function นี้แทนใน Service
func ToDomain_CreateProduct(req *dto.CreateProductReq) *domain.Product {
	categories := make([]domain.Category, len(req.CategoryIDs))
	for i, id := range req.CategoryIDs {
		categories[i] = domain.Category{ID: id}
	}
	variants := make([]domain.ProductVariant, len(req.Variants))
	for i, vReq := range req.Variants {

		attributes := make([]domain.VariantAttribute, len(vReq.AttributeValueIDs))
		for j, attrID := range vReq.AttributeValueIDs {
			attributes[j] = domain.VariantAttribute{ID: attrID}
		}

		variants[i] = domain.ProductVariant{
			NameVariant: vReq.Name,
			Sku:         vReq.Sku,
			Price:       vReq.Price,
			Stock:       vReq.Stock,
			IsActive:    true,
			Attributes:  attributes,
			// ID: 0 (Database will generate)
			// ProductID: 0 (GORM will map when saving Root)
		}
	}

	return &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		ImageURLs:   req.ImageURLs,
		Categories:  categories,
		Variants:    variants,
	}
}

// ToDomain_Variants แปลง CreateVariantReq เป็น Domain Slice
//
// KEY PATTERN: VariantAttribute ใส่แค่ ID (ไม่ใส่ Name/Value)
// WHY?
//   - GORM ใช้ ID ในการผูก many2many relation ของ variant_values join table
//   - ถ้าใส่ struct เต็ม GORM จะ upsert attribute_values แทนที่จะ link relation
//   - ID-only struct → GORM ไป look up existing record แล้วสร้างแค่ join
func ToDomain_Variants(reqVariants []dto.CreateVariantReq) []domain.ProductVariant {
	variants := make([]domain.ProductVariant, len(reqVariants))

	for i, v := range reqVariants {
		// Map Attribute IDs ให้กลายเป็น Domain Struct ที่มีแค่ ID
		attributes := make([]domain.VariantAttribute, len(v.AttributeValueIDs))
		for j, id := range v.AttributeValueIDs {
			attributes[j] = domain.VariantAttribute{
				ID: id, // <-- ใส่แค่ ID ตรงนี้ เดี๋ยว GORM เอาไปผูก Relation ให้เอง!
			}
		}

		// ประกอบร่าง Variant
		variants[i] = domain.ProductVariant{
			Sku:         v.Sku,
			NameVariant: v.Name,
			Price:       v.Price,
			Stock:       v.Stock,
			IsActive:    true,
			Attributes:  attributes, // ใส่ Attributes ที่มีแค่ ID ลงไป
		}
	}
	return variants
}

// ToDomain_Categories แปลง CategoryIDs เป็น Domain Slice
func ToDomain_Categories(categoryIDs []uint) []domain.Category {
	categories := make([]domain.Category, len(categoryIDs))
	for i, id := range categoryIDs {
		categories[i] = domain.Category{ID: id}
	}
	return categories
}

// ToProductRes แปลง Domain Product เป็น Response DTO สำหรับ HTTP response
//
// WHY แยก Mapper ออกมาจาก Service?
//   - Domain Object นำไป return เป็น HTTP response โดยตรงไม่ได้
//     (Domain อาจมี field ที่ไม่ควร expose เช่น domainEvents, internal fields)
//   - DTO ควบคุม API contract แยกจาก Domain model
//
// NOTE: Categories map เป็น []ProductCategoryRes (ID + Name) เพื่อให้ Frontend ใช้งานได้ครบโดยไม่ต้อง call เพิ่ม
func ToProductRes(product *domain.Product) *dto.ProductRes {

	categories := make([]dto.ProductCategoryRes, len(product.Categories))
	variant := make([]dto.ProductVariantRes, len(product.Variants))

	for i, c := range product.Categories {
		categories[i] = dto.ProductCategoryRes{
			ID:   c.ID,
			Name: c.Name,
		}
	}

	for i, v := range product.Variants {
		option := make([]dto.VariantOptionRes, len(v.Attributes))

		for j, attr := range v.Attributes {
			option[j] = dto.VariantOptionRes{
				Name:  attr.Name,
				Value: attr.Value,
			}
		}

		variant[i] = dto.ProductVariantRes{
			ID:        v.ID,
			Sku:       v.Sku,
			Name:      v.NameVariant,
			Price:     v.Price,
			Stock:     v.Stock,
			IsActive:  v.IsActive,
			ImageUrls: v.ImageURLs,
			Options:   option,
		}
	}

	return &dto.ProductRes{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		ImageUrls:   product.ImageURLs,
		IsActive:    product.IsActive,
		Variants:    variant,
		Categories:  categories,
		CreatedBy:   product.CreatedBy,
	}
}
