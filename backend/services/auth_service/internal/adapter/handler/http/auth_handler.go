package http

import (
	service "auth_service/internal/core/port/service"
	req "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/gofiber/fiber/v2"
)

type authHandler struct {
	userSvc service.UserService
	// adminSvc service.AdminService
	authSvc service.AuthService
}

// func NewAuthHandler(userSvc service.UserService, adminSvc service.AdminService, authSvc service.AuthService) *authHandler {
// 	return &authHandler{userSvc: userSvc, adminSvc: adminSvc, authSvc: authSvc}
// }

func NewAuthHandler(userSvc service.UserService, authSvc service.AuthService) *authHandler {
	return &authHandler{userSvc: userSvc, authSvc: authSvc}
}

func (h *authHandler) Login(c *fiber.Ctx) error {
	req := &req.LoginRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	tokens, err := h.authSvc.LoginUser(c.Context(), req.Email, req.Password, c.IP(), req.DeviceInfo)

	if err != nil {
		// Map Domain Error to HTTP Error
		// (ตรงนี้อาจจะทำ Middleware Error Handler แยกก็ได้ครับ)
		return httpcore.HandleError(c, err) // ให้ Global Error Handler จัดการ หรือ map เองตรงนี้
	}

	return c.JSON(tokens)
}

func (h *authHandler) RefreshToken(c *fiber.Ctx) error {
	// สร้าง Request Struct สำหรับรับ refresh_token
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
	req := &RefreshRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	tokens, err := h.authSvc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(tokens)
}
