package http

import (
	"errs"
	"httpcore"
	dto "product_service/internal/core/port/service/dto"
	port "product_service/internal/core/port/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// attributeHandler ทำหน้าที่เป็น HTTP Adapter สำหรับ Attribute และ AttributeValue
// รวม 2 resource ไว้ใน handler เดียวเพราะ AttributeValue ไม่มี lifecycle อิสระ
// มันต้องผูกอยู่กับ Attribute เสมอ (route: /attributes/:id/values)
type attributeHandler struct {
	cmdService   port.AttributeCommandService
	queryService port.AttributeQueryService
	validator    *validator.Validate
}

func NewAttributeHandler(cmd port.AttributeCommandService, query port.AttributeQueryService) *attributeHandler {
	return &attributeHandler{
		cmdService:   cmd,
		queryService: query,
		validator:    validator.New(),
	}
}

// GET /attributes
// @Summary List all attributes with their values
// @Description Returns all attributes (e.g. Color, Size) each populated with their possible values.
// @Tags Attributes (Query)
// @Produce json
// @Success 200 {array} dto.AttributeRes "Attribute list"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /attributes [get]
func (h *attributeHandler) ListAttributes(c *fiber.Ctx) error {
	attrs, err := h.queryService.GetAllAttributes(c.UserContext())
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(attrs)
}

// GET /attributes/:id
// @Summary Get attribute by ID
// @Description Returns a single attribute with all its values.
// @Tags Attributes (Query)
// @Produce json
// @Param id path int true "Attribute ID"
// @Success 200 {object} dto.AttributeRes "Attribute detail"
// @Failure 404 {object} map[string]interface{} "Attribute not found"
// @Router /attributes/{id} [get]
func (h *attributeHandler) GetAttribute(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid attribute ID"))
	}

	attr, err := h.queryService.GetAttributeByID(c.UserContext(), uint(id))
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(attr)
}

// POST /attributes/admin
// @Summary Create a new attribute (Admin)
// @Description Creates a new attribute type such as Color or Size.
// @Tags Attributes (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateAttributeReq true "Attribute data"
// @Success 201 {object} map[string]string "Attribute created successfully"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /attributes/admin [post]
func (h *attributeHandler) CreateAttribute(c *fiber.Ctx) error {
	req := &dto.CreateAttributeReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	if err := h.cmdService.CreateAttribute(c.UserContext(), req); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Attribute created successfully"})
}

// PUT /attributes/admin/:id
// @Summary Update an attribute (Admin)
// @Description Updates the name of an existing attribute.
// @Tags Attributes (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Attribute ID"
// @Param body body dto.UpdateAttributeReq true "Updated attribute data"
// @Success 200 {object} map[string]string "Attribute updated successfully"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 404 {object} map[string]interface{} "Attribute not found"
// @Router /attributes/admin/{id} [put]
func (h *attributeHandler) UpdateAttribute(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid attribute ID"))
	}

	req := &dto.UpdateAttributeReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}
	// Override ด้วย URL param เพื่อป้องกัน id ใน body ไม่ตรงกับ URL (path param takes precedence)
	req.AttributeID = uint(id)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	if err := h.cmdService.UpdateAttribute(c.UserContext(), req); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Attribute updated successfully"})
}

// DELETE /attributes/admin/:id
// @Summary Delete an attribute (Admin)
// @Description Deletes an attribute and cascades to its values.
// @Tags Attributes (Command)
// @Produce json
// @Security BearerAuth
// @Param id path int true "Attribute ID"
// @Success 200 {object} map[string]string "Attribute deleted successfully"
// @Failure 404 {object} map[string]interface{} "Attribute not found"
// @Router /attributes/admin/{id} [delete]
func (h *attributeHandler) DeleteAttribute(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid attribute ID"))
	}

	if err := h.cmdService.DeleteAttribute(c.UserContext(), uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Attribute deleted successfully"})
}

// POST /attributes/admin/:id/values
// @Summary Add a value to an attribute (Admin)
// @Description Adds a new possible value (e.g. "Red") to an existing attribute (e.g. "Color").
// @Tags Attributes (Command)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Attribute ID"
// @Param body body dto.CreateAttributeValueReq true "Attribute value data"
// @Success 201 {object} map[string]string "Attribute value created successfully"
// @Failure 400 {object} map[string]interface{} "Validation error"
// @Failure 404 {object} map[string]interface{} "Attribute not found"
// @Router /attributes/admin/{id}/values [post]
func (h *attributeHandler) CreateAttributeValue(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid attribute ID"))
	}

	req := &dto.CreateAttributeValueReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body"))
	}
	// Bind AttributeID จาก URL path (:id) เพื่อให้ client ไม่ต้องส่ง attribute_id ใน body ซ้ำซ้อน
	// และเพื่อป้องกัน mismatch ระหว่าง path กับ body
	req.AttributeID = uint(id)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	if err := h.cmdService.CreateAttributeValue(c.UserContext(), req); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Attribute value created successfully"})
}

// DELETE /attributes/admin/:id/values/:valueId
// @Summary Delete an attribute value (Admin)
// @Description Removes a specific value from an attribute.
// @Tags Attributes (Command)
// @Produce json
// @Security BearerAuth
// @Param id path int true "Attribute ID"
// @Param valueId path int true "Attribute Value ID"
// @Success 200 {object} map[string]string "Attribute value deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid ID"
// @Router /attributes/admin/{id}/values/{valueId} [delete]
func (h *attributeHandler) DeleteAttributeValue(c *fiber.Ctx) error {
	valueId, err := c.ParamsInt("valueId")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid attribute value ID"))
	}

	if err := h.cmdService.DeleteAttributeValue(c.UserContext(), uint(valueId)); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Attribute value deleted successfully"})
}
