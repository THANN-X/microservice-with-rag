package http

import (
	"errs"
	"httpcore"
	service "product_service/internal/core/port/service"

	dto "product_service/internal/core/port/service/dto"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type productHandler struct {
	cmdService   service.ProductCommandService
	queryService service.ProductQueryService
	validator    *validator.Validate
}

func NewProductHandler(cmd service.ProductCommandService, query service.ProductQueryService) *productHandler {
	return &productHandler{
		cmdService:   cmd,
		queryService: query,
		validator:    validator.New(),
	}
}

// POST /products
// CreateProduct สร้างสินค้าใหม่ (Admin เท่านั้น)
// @Summary Create a new product (Admin)
// @Description Create a new product with its variants and categories. Emits Outbox event.
// @Tags Products (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateProductReq true "Product Creation Data"
// @Success 201 {object} map[string]string "Product created successfully"
// @Failure 400 {object} map[string]interface{} "Validation error or invalid JSON"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/admin [post]
func (h *productHandler) CreateProduct(c *fiber.Ctx) error {
	req := &dto.CreateProductReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	// ดึงข้อมูล User ที่ Login อยู่ (จาก Middleware)
	requesterID, ok := c.Locals("user_id").(uint) // Type Assertion ให้ตรงกับตอน Set
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	// ใช้ c.UserContext() เพื่อรองรับ Tracing/Timeout จาก Fiber
	if err := h.cmdService.CreateProduct(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Product created successfully",
	})
}

// UpdateGeneralInfo อัปเดตข้อมูลทั่วไปของสินค้า
// @Summary Update product general info (Admin)
// @Description Update a product's name, description, and category associations. Emits PRODUCT_INFO_UPDATED outbox event.
// @Tags Products (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param body body dto.UpdateProductGeneralInfoReq true "Updated product data"
// @Success 200 {object} map[string]string "Product general info updated successfully"
// @Failure 400 {object} map[string]interface{} "Validation error or invalid ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/admin/{id}/general-info [put]
func (h *productHandler) UpdateGeneralInfo(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	req := &dto.UpdateProductGeneralInfoReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	// ProductID from URL takes precedence over any value in the request body
	req.ProductID = uint(id)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.UpdateProductGeneralInfo(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Product general info updated successfully",
	})
}

// UpdateVariantPrice อัปเดตราคา Variant
// @Summary Update variant price (Admin)
// @Description Update the price of a specific product variant. Emits PRODUCT_PRICE_CHANGED outbox event.
// @Tags Variants (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param variantId path int true "Variant ID"
// @Param body body dto.UpdateVariantPriceReq true "New price data"
// @Success 200 {object} map[string]string "Product variant price updated successfully"
// @Failure 400 {object} map[string]interface{} "Validation error or invalid ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Product or variant not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/admin/{id}/variants/{variantId}/price [patch]
func (h *productHandler) UpdateVariantPrice(c *fiber.Ctx) error {
	// Parse IDs จาก URL (Product ID และ Variant ID)
	// สมมติ Route คือ PATCH /products/:id/variants/:variantId/price
	pId, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	vId, err := c.ParamsInt("variantId")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid variant ID"))
	}

	req := &dto.UpdateVariantPriceReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	// IDs from URL take precedence over any value in the request body
	req.ProductID = uint(pId)
	req.VariantID = uint(vId)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	requesterID, ok := c.Locals("user_id").(uint)

	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.UpdateVariantPrice(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Product variant price updated successfully",
	})
}

// GET /products/:id
// GetProduct ดึงข้อมูลสินค้า 1 ตัวตาม ID
// @Summary Get product by ID
// @Description Retrieve detailed information of a specific product including variants and categories.
// @Tags Products (Query)
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} dto.ProductRes "Product details"
// @Failure 400 {object} map[string]interface{} "Invalid ID"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{id} [get]
func (h *productHandler) GetProduct(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	product, err := h.queryService.GetProductByID(c.UserContext(), uint(id))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	// ถ้า Service return nil แสดงว่าไม่เจอ (หรือ service อาจจะ return NotFoundError มาแล้ว)
	if product == nil {
		return httpcore.HandleError(c, errs.NewNotFoundError("Product not found"))
	}

	return c.Status(fiber.StatusOK).JSON(product)
}

// POST /products/:id/variants
// AddVariant เพิ่ม Variant ใหม่ให้สินค้า
// @Summary Add a new variant to an existing product
// @Description Adds a new color/size (variant) to an existing product and emits an outbox event.
// @Tags Variants (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param body body dto.AddVariantReq true "Variant Data"
// @Success 201 {object} map[string]string "Variant added successfully"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Router /products/admin/{id}/variants [post]
func (h *productHandler) AddVariant(c *fiber.Ctx) error {
	pId, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	req := &dto.AddVariantReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	req.ProductID = uint(pId) // Override ID

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.AddVariant(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Variant added successfully",
	})
}

// DeleteProduct ลบสินค้า (Soft Delete)
// @Summary Delete a product (Admin)
// @Description Soft-delete a product by ID. Emits PRODUCT_DELETED outbox event.
// @Tags Products (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} map[string]string "Product deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/admin/{id} [delete]
func (h *productHandler) DeleteProduct(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.DeleteProduct(c.UserContext(), requesterID, uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Product deleted successfully",
	})
}

// AdjustStock ปรับสต็อก Variant (Admin)
// @Summary Adjust variant stock (Admin)
// @Description Manually set the absolute stock value for a variant (stock take / damage write-off). Emits STOCK_ADJUSTED outbox event.
// @Tags Variants (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param variantId path int true "Variant ID"
// @Param body body dto.AdjustStockReq true "Stock adjustment data"
// @Success 200 {object} map[string]string "Stock adjusted successfully"
// @Failure 400 {object} map[string]interface{} "Validation error or invalid ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/admin/{id}/variants/{variantId}/stock [patch]
func (h *productHandler) AdjustStock(c *fiber.Ctx) error {
	pId, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	vId, err := c.ParamsInt("variantId")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid variant ID"))
	}

	req := &dto.AdjustStockReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	req.ProductID = uint(pId)
	req.VariantID = uint(vId)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.AdjustStock(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Stock adjusted successfully",
	})
}

// GET /products (หรือ /api/v1/products)
// ListProducts ดึงรายการสินค้าทั้งหมด
// @Summary List all products (Admin & Public)
// @Description Get a list of products with pagination, filtering by category, and searching.
// @Tags Products (Query)
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by product name or SKU"
// @Param category query int false "Filter by Category ID"
// @Param is_active query bool false "Filter by active status (true/false)"
// @Param sort_by query string false "Sort field (e.g., created_at, name, price)" default(created_at)
// @Param order query string false "Sort order (asc or desc)" default(desc)
// @Success 200 {object} dto.ProductListRes "Successfully retrieved product list"
// @Failure 400 {object} map[string]interface{} "Invalid query parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products [get]
func (h *productHandler) ListProducts(c *fiber.Ctx) error {
	req := new(dto.ListProductReq)

	/* ดึงค่าจาก URL Query String (?page=1&limit=10) มาใส่ Struct อัตโนมัติ
	ใช้ QueryParser ของ Fiber เพื่อดึงค่าจาก URL เช่น ?page=1&limit=10&search=shirt
	 มาใส่ใน Struct req ให้อัตโนมัติ (ตาม tag `query:"..."` ที่เราตั้งไว้)*/
	if err := c.QueryParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid query parameters"))
	}

	// (Optional) Validate ค่าซักหน่อย เผื่อใส่ Limit มาเกิน 100
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	// เรียกใช้งาน Query Service
	//ใช้ c.UserContext() เพื่อส่ง Context ของ Request เข้าไปเผื่อมี Timeout/Tracing
	result, err := h.queryService.ListProducts(c.UserContext(), req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// PATCH /products/admin/:id/active
// SetProductActive เปิด/ปิดการแสดงสินค้า (Admin เท่านั้น)
// @Summary Activate or deactivate a product (Admin)
// @Description Toggle product visibility. is_active=false hides the product from the storefront.
// @Tags Products (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param body body dto.SetProductActiveReq true "Active status"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/admin/{id}/active [patch]
func (h *productHandler) SetProductActive(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	req := &dto.SetProductActiveReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	req.ProductID = uint(id)

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.SetProductActive(c.UserContext(), requesterID, req.ProductID, req.IsActive); err != nil {
		return httpcore.HandleError(c, err)
	}

	msg := "Product deactivated successfully"

	if req.IsActive {
		msg = "Product activated successfully"
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": msg})
}

// PATCH /products/admin/:id/variants/:variantId/active
// SetVariantActive เปิด/ปิดการแสดง Variant เฉพาะตัว (Admin เท่านั้น)
// @Summary Activate or deactivate a product variant (Admin)
// @Description Toggle variant visibility without affecting the parent product.
// @Tags Products (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param variantId path int true "Variant ID"
// @Param body body dto.SetVariantActiveReq true "Active status"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/admin/{id}/variants/{variantId}/active [patch]
func (h *productHandler) SetVariantActive(c *fiber.Ctx) error {
	pId, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	vId, err := c.ParamsInt("variantId")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid variant ID"))
	}

	req := &dto.SetVariantActiveReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	req.ProductID = uint(pId)
	req.VariantID = uint(vId)

	requesterID, ok := c.Locals("user_id").(uint)

	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.SetVariantActive(c.UserContext(), requesterID, req.ProductID, req.VariantID, req.IsActive); err != nil {
		return httpcore.HandleError(c, err)
	}

	msg := "Variant deactivated successfully"

	if req.IsActive {
		msg = "Variant activated successfully"
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": msg})
}

// PATCH /products/admin/:id/images
// UpdateProductImages แทนที่รูปภาพของ Product ทั้งหมด (Admin เท่านั้น)
// @Summary Replace product-level images (Admin)
// @Description Replace the full image list of a product with a new set of URLs.
// @Tags Products (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param body body dto.UpdateProductImagesReq true "Image URLs"
// @Success 200 {object} map[string]string "Product images updated successfully"
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/admin/{id}/images [patch]
func (h *productHandler) UpdateProductImages(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	req := &dto.UpdateProductImagesReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	req.ProductID = uint(id)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.UpdateProductImages(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Product images updated successfully"})
}

// PATCH /products/admin/:id/variants/:variantId/images
// UpdateVariantImages แทนที่รูปภาพของ Variant เฉพาะตัว (Admin เท่านั้น)
// @Summary Replace variant images (Admin)
// @Description Replace the full image list of a specific variant (e.g. colour-specific photos).
// @Tags Variants (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param variantId path int true "Variant ID"
// @Param body body dto.UpdateVariantImagesReq true "Image URLs"
// @Success 200 {object} map[string]string "Variant images updated successfully"
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/admin/{id}/variants/{variantId}/images [patch]
func (h *productHandler) UpdateVariantImages(c *fiber.Ctx) error {
	pId, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid product ID"))
	}

	vId, err := c.ParamsInt("variantId")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid variant ID"))
	}

	req := &dto.UpdateVariantImagesReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}

	req.ProductID = uint(pId)
	req.VariantID = uint(vId)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid user ID in token"))
	}

	if err := h.cmdService.UpdateVariantImages(c.UserContext(), requesterID, req); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Variant images updated successfully"})
}
