package http

import (
	service "auth_service/internal/core/port/service"
	req "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// What: adminHandler จัดการ HTTP request เกี่ยวกับ admin account
// Why:  แยกออกจาก userHandler เพราะ admin ใช้ authentication flow ต่างกัน (username แทน email)
type adminHandler struct {
	adminSvc  service.AdminService
	validater *validator.Validate
}

// What: constructor — inject adminService
func NewAdminHandler(adminSvc service.AdminService) *adminHandler {
	return &adminHandler{adminSvc: adminSvc, validater: validator.New()}
}

// What: สร้าง admin ใหม่ — endpoint นี้ถูกกั้นด้วย adminSecretGuard middleware ก่อนเสมอ
// Why:  admin ไม่ควรสร้างได้เองโดยไม่ยืนยันตัวตนกับระบบ
func (h *adminHandler) RegisterAdmin(c *fiber.Ctx) error {
	// What: parse JSON body → CreateAdminRequest
	req := &req.CreateAdminRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validater.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	// What: เรียก service สร้าง admin โดยส่ง password แยกออกมา
	// Why: service จะ hash password เอง ไม่ใช่ handler
	newAdmin, err := h.adminSvc.RegisterAdmin(c.Context(), req, req.Password)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(newAdmin)
}
