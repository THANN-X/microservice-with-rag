package http

import (
	port "cart_service/internal/core/port/service"
	"cart_service/internal/core/port/service/dto"
	"errs"
	"httpcore"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type cartHandler struct {
	cmdService   port.CartCommandService
	queryService port.CartQueryService
	validator    *validator.Validate
}

func NewCartHandler(cmd port.CartCommandService, query port.CartQueryService) *cartHandler {
	return &cartHandler{
		cmdService:   cmd,
		queryService: query,
		validator:    validator.New(),
	}
}

// GET /cart
// GetCart คืนตะกร้าของ user ที่ login อยู่
// ถ้ายังไม่มีตะกร้าจะคืน empty cart (ไม่ 404)
func (h *cartHandler) GetCart(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	res, err := h.queryService.GetCart(c.UserContext(), userID)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// POST /cart/items
// AddItem เพิ่ม item เข้าตะกร้า (Lazy Creation: สร้างตะกร้าถ้ายังไม่มี)
func (h *cartHandler) AddItem(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	req := &dto.AddCartItemReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	res, err := h.cmdService.AddItem(c.UserContext(), userID, req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// PUT /cart/items/:variantId
// UpdateItemQuantity อัพเดตจำนวน item (quantity=0 จะลบ item ออก)
func (h *cartHandler) UpdateItemQuantity(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	variantIDStr := c.Params("variantId")
	variantID64, err := strconv.ParseUint(variantIDStr, 10, 32)
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid variant ID"))
	}

	req := &dto.UpdateCartItemReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid request body"))
	}
	req.VariantID = uint(variantID64)

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	res, err := h.cmdService.UpdateItemQuantity(c.UserContext(), userID, req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// DELETE /cart/items/:variantId
// RemoveItem ลบ item ออกจากตะกร้า
func (h *cartHandler) RemoveItem(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	variantIDStr := c.Params("variantId")
	variantID64, err := strconv.ParseUint(variantIDStr, 10, 32)
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid variant ID"))
	}

	res, err := h.cmdService.RemoveItem(c.UserContext(), userID, uint(variantID64))
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// DELETE /cart
// ClearCart ล้างตะกร้าทั้งหมด
func (h *cartHandler) ClearCart(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	if err := h.cmdService.ClearCart(c.UserContext(), userID); err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
