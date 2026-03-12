package http

import (
	service "auth_service/internal/core/port/service"
	req "auth_service/internal/core/port/service/dto"
	"errs"
	"httpcore"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type userHandler struct {
	// Handler fields and methods
	userSvc   service.UserService
	validator *validator.Validate
}

// Constructor for UserHandler
func NewUserHandler(userSvc service.UserService) *userHandler {
	return &userHandler{userSvc: userSvc, validator: validator.New()}
}

// Handler for getting user profile
func (h *userHandler) GetUserProfile(c *fiber.Ctx) error {
	/* Handler logic for getting user profile
	// id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	// if err != nil {
	// 	return handleError(c, errs.NewValidationError("Invalid user ID format"))
	// }*/

	// Parse ID
	targetID, err := c.ParamsInt("id")
	if err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid user ID format"))
	}

	// Check Permission
	if err := h.checkOwnerOrAdmin(c, uint(targetID)); err != nil {
		return httpcore.HandleError(c, err)
	}

	// Call Service
	// Call service to get user profile
	user, err := h.userSvc.GetUserProfile(c.Context(), uint(targetID))
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

// Handler for getting own profile
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
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
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

	// Check Permission
	if err := h.checkOwnerOrAdmin(c, uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}

	// Call service to change password
	req := &req.ChangePasswordReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

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

	// Check Permission
	if err := h.checkOwnerOrAdmin(c, uint(id)); err != nil {
		return httpcore.HandleError(c, err)
	}

	// Call service to update user profile
	req := &req.UpdateUserRequest{}
	// Parse request body
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	// Call service to update user profile
	user, err := h.userSvc.UpdateUserInfo(c.Context(), uint(id), req)
	// Handle errors
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success", "data": user})
}

// checkOwnerOrAdmin: เช็คว่าเป็นเจ้าของ ID หรือเป็น Admin
func (h *userHandler) checkOwnerOrAdmin(c *fiber.Ctx, targetID uint) error {
	// ดึงข้อมูลคนเรียก
	requesterID, ok := c.Locals("user_id").(uint)
	if !ok {
		return errs.NewUnauthorizedError("User context missing")
	}

	requesterRole, ok := c.Locals("role").(string)
	if !ok {
		return errs.NewUnauthorizedError("User role missing")
	}

	// Logic: ถ้าเป็น Admin ให้ผ่านเลย
	if requesterRole == "admin" {
		return nil
	}

	// Logic: ถ้าไม่ใช่ Admin ต้องเป็น ID ตัวเองเท่านั้น
	if requesterID != targetID {
		return errs.NewForbiddenError("You usually don't have permission to access this resource")
	}

	return nil
}
