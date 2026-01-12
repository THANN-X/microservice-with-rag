package http

import (
	service "auth_service/internal/core/port/service"
	req "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/gofiber/fiber/v2"
)

type userHandler struct {
	// Handler fields and methods
	userSvc service.UserService
}

// Constructor for UserHandler
func NewUserHandler(userSvc service.UserService) *userHandler {
	return &userHandler{userSvc: userSvc}
}

// Handler for getting user profile
func (h *userHandler) GetUserProfile(c *fiber.Ctx) error {
	// Handler logic for getting user profile
	// id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	// if err != nil {
	// 	return handleError(c, errs.NewValidationError("Invalid user ID format"))
	// }

	// 1. ดึงข้อมูล User ที่ Login อยู่ (จาก Middleware)
	requesterID := c.Locals("user_id").(uint)
	requesterRole := c.Locals("role").(string)

	// 2. ดึง ID ที่ต้องการดูจาก URL (/users/:id)
	targetID, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}

	// 3. Logic ตรวจสอบสิทธิ์ (Authorization)
	// อนุญาตถ้า: (Role เป็น "admin") หรือ (ID ที่ขอดู == ID ของตัวเอง)
	if requesterRole != "admin" && uint(targetID) != requesterID {
		return httpcore.HandleError(c, errs.NewValidationError("You don't have permission to access this resource"))
	}

	// 4. เรียก Service เพื่อดึงข้อมูล (reuse service เดิมได้เลย)
	// Call service to get user profile
	user, err := h.userSvc.GetUserProfile(c.Context(), uint(targetID))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

func (h *userHandler) GetMyProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	user, err := h.userSvc.GetUserProfile(c.Context(), uint(userID))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(user)

}

// Handler for user registration
func (h *userHandler) RegisterUser(c *fiber.Ctx) error {
	// Handler logic for user registration
	req := &req.CreateUserRequest{}
	if err := c.BodyParser(&req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	// Call service to register user
	newUser, err := h.userSvc.RegisterNewUser(c.Context(), req, req.Password)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(newUser)
}

// Handler for changing password
func (h *userHandler) ChangePassword(c *fiber.Ctx) error {
	// Handler logic for changing password
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}
	// Call service to change password
	req := &req.ChangePasswordReq{}
	if err := c.BodyParser(&req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}
	// Call service to change password
	err = h.userSvc.UpdatePassword(c.Context(), uint(id), req.OldPassword, req.NewPassword)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// Handler for updating user profile
func (h *userHandler) UpdateUserProfile(c *fiber.Ctx) error {
	// Handler logic for updating user profile
	id, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}
	// Call service to update user profile
	req := &req.UpdateUserRequest{}
	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	// Call service to update user profile
	user, err := h.userSvc.UpdateUserInfo(c.Context(), uint(id), req)
	// Handle errors
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": user})
}
