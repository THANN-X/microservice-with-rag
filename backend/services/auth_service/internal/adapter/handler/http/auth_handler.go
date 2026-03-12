package http

import (
	service "auth_service/internal/core/port/service"
	dto "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type authHandler struct {
	userSvc   service.UserService
	adminSvc  service.AdminService
	authSvc   service.AuthService
	validator *validator.Validate
}

func NewAuthHandler(userSvc service.UserService, adminSvc service.AdminService, authSvc service.AuthService) *authHandler {
	return &authHandler{userSvc: userSvc, adminSvc: adminSvc, authSvc: authSvc, validator: validator.New()}
}

func (h *authHandler) LoginUser(c *fiber.Ctx) error {
	req := &dto.LoginRequest{}

	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	tokens, err := h.authSvc.LoginUser(c.Context(), req.Email, req.Password, c.IP(), req.DeviceInfo)

	if err != nil {
		// Map Domain Error to HTTP Error
		// (ตรงนี้อาจจะทำ Middleware Error Handler แยกก็ได้ครับ)
		return httpcore.HandleError(c, err) // ให้ Global Error Handler จัดการ หรือ map เองตรงนี้
	}

	return c.JSON(tokens)
}

func (h *authHandler) LoginAdmin(c *fiber.Ctx) error {
	req := &dto.LoginAdminRequest{}

	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	tokens, err := h.authSvc.LoginAdmin(c.Context(), req.Username, req.Password, c.IP(), req.DeviceInfo)

	if err != nil {
		/* Map Domain Error to HTTP Error
		(ตรงนี้อาจจะทำ Middleware Error Handler แยกก็ได้)*/

		return httpcore.HandleError(c, err) // ให้ Global Error Handler จัดการ หรือ map เอง
	}

	return c.JSON(tokens)
}

func (h *authHandler) Logout(c *fiber.Ctx) error {
	var req dto.LogoutRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "refresh_token is required",
		})
	}

	err := h.authSvc.Logout(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

func (h *authHandler) RefreshToken(c *fiber.Ctx) error {
	// สร้าง Request Struct สำหรับรับ refresh_token
	req := &dto.RefreshRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	tokens, err := h.authSvc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(tokens)
}
