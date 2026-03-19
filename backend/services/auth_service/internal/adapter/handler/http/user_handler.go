package http

import (
	service "auth_service/internal/core/port/service"
	req "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// What: userHandler เป็น HTTP adapter รับ request และให้ userService ประมวลผล
// Why:  handler ควรบาง — parse request, validate, เรียก service, ตอบกลับ response
//
//	business logic ทั้งหมดอยู่ใน service layer
type userHandler struct {
	userSvc   service.UserService
	validator *validator.Validate
}

// What: constructor — inject service และเตรียม validator instance
func NewUserHandler(userSvc service.UserService) *userHandler {
	return &userHandler{userSvc: userSvc, validator: validator.New()}
}

// What: ดึงโปรไฟล์ user ตาม /:id — เฉพาะเจ้าของหรือ admin เท่านั้น
func (h *userHandler) GetProfile(c *fiber.Ctx) error {
	// What: parse :id จาก URL param — ต้องเป็น integer ที่ถูกต้อง
	targetID, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}

	// What: ตรวจสอบว่าเป็นเจ้าของ ID หรือ admin ก่อนดาเนินการ
	if err := h.checkOwnerOrAdmin(c, uint(targetID)); err != nil {
		return httpcore.HandleError(c, err)
	}

	// What: เรียก service ดึงโปรไฟล์
	user, err := h.userSvc.GetProfile(c.Context(), uint(targetID))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// What: ดึงโปรไฟล์ของตัวเอง — ID อ่านจาก JWT claims (ไม่ต้องส่ง ID ใน URL)
// Why:  ป้องกัน user อ่าน profile คนอื่นโดยบิด ID ใน URL
func (h *userHandler) GetMyProfile(c *fiber.Ctx) error {
	// What: ดึง user_id จาก context ที่ authMiddleware inject ไว้
	userID := c.Locals("user_id").(uint)

	user, err := h.userSvc.GetProfile(c.Context(), uint(userID))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// What: สมัคร user ใหม่ — endpoint นี้เปิด public ไม่ต้อง login
func (h *userHandler) RegisterUser(c *fiber.Ctx) error {
	// What: parse JSON body → CreateUserRequest struct
	req := &req.CreateUserRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	// What: validate field rules (เช่น required, min=2, email format)
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	newUser, err := h.userSvc.RegisterUser(c.Context(), req, req.Password)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(newUser)
}

// What: เปลี่ยน password — ตองส่ง old password + new password
func (h *userHandler) ChangePassword(c *fiber.Ctx) error {
	// What: parse :id จาก URL
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}

	// What: เฉพาะเจ้าของหรือ admin เท่านั้นที่เปลี่ยน password ได้
	if err := h.checkOwnerOrAdmin(c, uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}

	// What: parse body ถ้าผ่านแล้วค่อย parse body (ลำดับสำคัญ — check permission ก่อน)
	req := &req.ChangePasswordReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	err = h.userSvc.ChangePassword(c.Context(), uint(id), req.OldPassword, req.NewPassword)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// What: อัปเดต profile (ชื่อ, เบอร์, ที่อยู่) — ไม่รวม email/password
func (h *userHandler) UpdateProfile(c *fiber.Ctx) error {
	// What: parse :id จาก URL
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}

	// What: check permission ก่อน parse body — fail fast เร็วถ้าไม่มีสิทธิ์
	if err := h.checkOwnerOrAdmin(c, uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}

	req := &req.UpdateUserRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	user, err := h.userSvc.UpdateUserProfile(c.Context(), uint(id), req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": user})
}

// What: authorization check — อนุญาตเฉพาะเจ้าของ ID หรือ role admin
// Why:  รวม logic นี้ไว้ในเมธอดเดียวเพื่อ reuse หลาย handler โดยไม่ต้องทำซ้ำในทุกจุด
func (h *userHandler) checkOwnerOrAdmin(c *fiber.Ctx, targetID uint) error {
	// What: ดึง user_id จาก JWT claims ที่ authMiddleware เตรียมไว้
	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return errs.NewUnauthorizedError("User context missing")
	}

	requesterRole, ok := c.Locals("role").(string)
	if !ok {
		return errs.NewUnauthorizedError("User role missing")
	}

	// What: admin ผ่านได้ทุก resource — superuser behavior
	if requesterRole == "admin" {
		return nil
	}

	// What: user ทั่วไปเข้าถึงได้เฉพาะ resource ของตัวเอง
	if requesterID != targetID {
		return errs.NewForbiddenError("You usually don't have permission to access this resource")
	}

	return nil
}
