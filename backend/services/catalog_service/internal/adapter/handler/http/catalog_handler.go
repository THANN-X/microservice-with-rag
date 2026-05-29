package http

import (
	serviceport "catalog_service/internal/core/port/service"
	"catalog_service/internal/core/port/service/dto"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"httpcore"
)

type catalogHandler struct {
	queryService serviceport.CatalogQueryService
	validator    *validator.Validate
}

func NewCatalogHandler(queryService serviceport.CatalogQueryService) *catalogHandler {
	return &catalogHandler{
		queryService: queryService,
		validator:    validator.New(),
	}
}

// SearchProducts godoc
// @Summary     List / search catalog products
// @Description ดึงรายการสินค้าสำหรับลูกค้า รองรับ full-text search, filter by category, pagination
// @Tags        catalog
// @Produce     json
// @Param       page        query int    false "Page number (default: 1)"
// @Param       limit       query int    false "Items per page (default: 10, max: 100)"
// @Param       search      query string false "Full-text search on name and description"
// @Param       category_id query int    false "Filter by category ID"
// @Param       sort_by     query string false "Sort field (default: created_at)"
// @Param       order       query string false "asc or desc (default: desc)"
// @Success     200 {object} dto.ProductListRes
// @Router      /catalog/products [get]
func (h *catalogHandler) SearchProducts(c *fiber.Ctx) error {
	req := &dto.SearchProductsReq{}
	if err := c.QueryParser(req); err != nil {
		return httpcore.HandleError(c, err)
	}

	result, err := h.queryService.SearchProducts(c.UserContext(), req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(result)
}

// GetProduct godoc
// @Summary     Get a single catalog product
// @Description ดึงข้อมูลสินค้าตาม product ID สำหรับหน้า product detail
// @Tags        catalog
// @Produce     json
// @Param       productId path int true "Product ID"
// @Success     200 {object} dto.CatalogProductRes
// @Failure     400 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /catalog/products/{productId} [get]
func (h *catalogHandler) GetProduct(c *fiber.Ctx) error {
	idStr := c.Params("productId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid product id"})
	}

	result, err := h.queryService.GetProductByID(c.UserContext(), uint(id))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(result)
}

// GetVariantInfo godoc
// @Summary     Get variant info for cart enrichment
// @Description ดึงข้อมูล variant แบบ flat (product_name, variant_name, price, image) ตาม variant ID
// @Tags        catalog
// @Produce     json
// @Param       variantId path int true "Variant ID"
// @Success     200 {object} dto.VariantInfoRes
// @Failure     400 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /catalog/variants/{variantId} [get]
func (h *catalogHandler) GetVariantInfo(c *fiber.Ctx) error {
	idStr := c.Params("variantId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid variant id"})
	}

	result, err := h.queryService.GetVariantInfo(c.UserContext(), uint(id))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(result)
}
