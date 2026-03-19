package http

import (
	service "auth_service/internal/core/port/service"
	dto "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// What: authHandler จัดการ HTTP request สำหรับ login, logout, refresh token
// Why:  แยก auth route ออกจาก user route เนื่องจาก auth เป็น cross-cutting concern
//       และเกี่ยวข้องกับทั้ง user และ admin
type authHandler struct {
	userSvc   service.UserService
	adminSvc  service.AdminService
	authSvc   service.AuthService
	validator *validator.Validate
}

// What: constructor — inject ทั้ง 3 services
func NewAuthHandler(userSvc service.UserService, adminSvc service.AdminService, authSvc service.AuthService) *authHandler {
	return &authHandler{userSvc: userSvc, adminSvc: adminSvc, authSvc: authSvc, validator: validator.New()}
}

// What: รับ login request ของ user (เอา email + password + device info)
//       แล้วคืน access_token + refresh_token กลับ
func (h *authHandler) LoginUser(c *fiber.Ctx) error {
	req := &dto.LoginRequest{}

	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	// What: ส่ง IP และ device info ไปด้วยเพื่อเก็บใน session
	tokens, err := h.authSvc.LoginUser(c.Context(), req.Email, req.Password, c.IP(), req.DeviceInfo)
	if err != nil {
		// What: ใช้ httpcore.HandleError เป็น centralized error mapper
		// TODO: หาก error mapping หลาย pattern แนะนำใช้ Fiber error handler middleware แทน
		return httpcore.HandleError(c, err)
	}

	return c.JSON(tokens)
}

// What: รับ login request ของ admin (เอา username + password)
//       แล้วคืน access_token + refresh_token กลับ
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
		return httpcore.HandleError(c, err)
	}

	return c.JSON(tokens)
}

// What: รับ refresh_token แล้ว revoke session นั้น (logout)
// Why:  ใช้ refresh_token แทน access_token เพราะ access token อายุสั้นเกินไปแล้ว ไม่มีประโยชน์ในการร้องขอ logout
func (h *authHandler) Logout(c *fiber.Ctx) error {
	var req dto.LogoutRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// What: validate ว่ามี refresh_token ส่งมา
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

// What: รับ refresh_token แล้วออก access_token ใหม่
// Why:  รองรับการ login ของ client โดยไม่ต้องใส่ credentials ใหม่ทุกครั้ง
func (h *authHandler) RefreshToken(c *fiber.Ctx) error {
	// What: parse body สำหรับรับ refresh_token
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
