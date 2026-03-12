package http

import (
	service "auth_service/internal/core/port/service"
	req "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type adminHandler struct {
	adminSvc  service.AdminService
	validater *validator.Validate
}

func NewAdminHandler(adminSvc service.AdminService) *adminHandler {
	return &adminHandler{adminSvc: adminSvc, validater: validator.New()}
}

// Handler for admin registration
func (h *adminHandler) CreateAdmin(c *fiber.Ctx) error {
	// Handler logic for admin registration
	req := &req.CreateAdminRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validater.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	// Call service to register admin
	newAdmin, err := h.adminSvc.CreateAdmin(c.Context(), req, req.Password)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(newAdmin)
}
