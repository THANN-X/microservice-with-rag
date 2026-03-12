package http

import (
	"errs"
	"httpcore"
	port "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// categoryHandler เป็น HTTP Adapter ที่เชื่อมระหว่าง HTTP (Fiber) กับ Service Layer
// ตาม Hexagonal Architecture Handler ควรทำแค่:
//  1. Parse + validate request
//  2. เรียก service
//  3. แปลง result เป็น HTTP response
//
// ไม่ควรมี business logic อยู่ใน Handler
type categoryHandler struct {
	cmdService   port.CategoryCommandService
	queryService port.CategoryQueryService
	// validator instance ถูกสร้างครั้งเดียวตอน init เพราะ validator.New() มี cost (reflect)
	// ไม่สร้างใหม่ทุก request
	validator *validator.Validate
}

func NewCategoryHandler(cmd port.CategoryCommandService, query port.CategoryQueryService) *categoryHandler {
	return &categoryHandler{
		cmdService:   cmd,
		queryService: query,
		validator:    validator.New(),
	}
}

// GET /categories
// @Summary List all categories
// @Description Returns all root categories with their children (tree structure).
// @Tags Categories (Query)
// @Produce json
// @Success 200 {array} dto.CategoryRes "Category tree"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /categories [get]
func (h *categoryHandler) ListCategories(c *fiber.Ctx) error {
	categories, err := h.queryService.GetAllCategories(c.UserContext())
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(categories)
}

// GET /categories/:id
// @Summary Get category by ID
// @Description Returns a single category with its children.
// @Tags Categories (Query)
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} dto.CategoryRes "Category detail"
// @Failure 404 {object} map[string]interface{} "Category not found"
// @Router /categories/{id} [get]
func (h *categoryHandler) GetCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid category ID"))
	}

	category, err := h.queryService.GetCategoryByID(c.UserContext(), uint(id))
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(category)
}

// POST /categories/admin
// @Summary Create a new category (Admin)
// @Description Creates a new product category. Supports nested categories via parent_id.
// @Tags Categories (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateCategoryReq true "Category data"
// @Success 201 {object} map[string]string "Category created successfully"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /categories/admin [post]
func (h *categoryHandler) CreateCategory(c *fiber.Ctx) error {
	req := &dto.CreateCategoryReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	if err := h.cmdService.CreateCategory(c.UserContext(), req); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Category created successfully"})
}

// PUT /categories/admin/:id
// @Summary Update a category (Admin)
// @Description Updates name, slug, description, is_active, and parent_id of a category.
// @Tags Categories (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Param body body dto.UpdateCategoryReq true "Updated category data"
// @Success 200 {object} map[string]string "Category updated successfully"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Category not found"
// @Router /categories/admin/{id} [put]
func (h *categoryHandler) UpdateCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid category ID"))
	}

	req := &dto.UpdateCategoryReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}
	// Override CategoryID ด้วย URL param เสมอ
	// ป้องกัน client ส่ง category_id ใน body ที่ต่างจาก URL (security: path param takes precedence)
	req.CategoryID = uint(id)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	if err := h.cmdService.UpdateCategory(c.UserContext(), req); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Category updated successfully"})
}

// DELETE /categories/admin/:id
// @Summary Delete a category (Admin)
// @Description Soft-deletes a category by ID.
// @Tags Categories (Command)
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Success 200 {object} map[string]string "Category deleted successfully"
// @Failure 404 {object} map[string]interface{} "Category not found"
// @Router /categories/admin/{id} [delete]
func (h *categoryHandler) DeleteCategory(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid category ID"))
	}

	if err := h.cmdService.DeleteCategory(c.UserContext(), uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Category deleted successfully"})
}

// PATCH /categories/admin/:id/active
// SetCategoryActive เปิด/ปิดการแสดง Category (Admin เท่านั้น)
// @Summary Activate or deactivate a category (Admin)
// @Description Toggle category visibility. is_active=false hides it from the storefront.
// @Tags Categories (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Param body body dto.SetCategoryActiveReq true "Active status"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /categories/admin/{id}/active [patch]
func (h *categoryHandler) SetCategoryActive(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid category ID"))
	}

	req := &dto.SetCategoryActiveReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}
	req.CategoryID = uint(id)

	if err := h.cmdService.SetCategoryActive(c.UserContext(), req.CategoryID, req.IsActive); err != nil {
		return httpcore.HandleError(c, err)
	}

	msg := "Category deactivated successfully"
	if req.IsActive {
		msg = "Category activated successfully"
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": msg})
}
