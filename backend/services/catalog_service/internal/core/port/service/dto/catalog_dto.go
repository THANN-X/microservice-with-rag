package dto

type SearchProductsReq struct {
	Page       int    `query:"page"`
	Limit      int    `query:"limit"`
	Search     string `query:"search"`
	CategoryID uint   `query:"category_id"`
	SortBy     string `query:"sort_by"`
	Order      string `query:"order"`
}

type CatalogProductRes struct {
	ProductID   uint                 `json:"product_id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	ImageURLs   []string             `json:"image_urls"`
	Categories  []CatalogCategoryRes `json:"categories"`
	Variants    []CatalogVariantRes  `json:"variants"`
	IsActive    bool                 `json:"is_active"`
}

type CatalogCategoryRes struct {
	CategoryID uint   `json:"category_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
}

type CatalogVariantRes struct {
	VariantID  uint                  `json:"variant_id"`
	Sku        string                `json:"sku"`
	Name       string                `json:"name"`
	Price      float64               `json:"price"`
	Stock      int                   `json:"stock"`
	IsActive   bool                  `json:"is_active"`
	ImageURLs  []string              `json:"image_urls"`
	Attributes []VariantAttributeRes `json:"attributes"`
}

type VariantAttributeRes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProductListRes struct {
	Items      []CatalogProductRes `json:"items"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

type VariantInfoRes struct {
	VariantID   uint     `json:"variant_id"`
	ProductID   uint     `json:"product_id"`
	ProductName string   `json:"product_name"`
	VariantName string   `json:"variant_name"`
	Price       float64  `json:"price"`
	ImageURL    string   `json:"image_url,omitempty"`
	ImageURLs   []string `json:"image_urls,omitempty"`
}
