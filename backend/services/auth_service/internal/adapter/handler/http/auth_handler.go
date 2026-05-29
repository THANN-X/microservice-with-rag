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
//
//	และเกี่ยวข้องกับทั้ง user และ admin
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

// setRefreshTokenCookie ตั้งค่า refresh_token เป็น HttpOnly cookie
// Why: refresh_token อยู่ใน HttpOnly cookie เพื่อป้องกัน XSS
//
//	Browser จะส่ง cookie อัตโนมัติ Next.js rewrite proxy จะ forward ให้ BFF → auth-service
func setRefreshTokenCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   false, // ตั้งเป็น true เมื่อ deploy บน HTTPS
		SameSite: "Lax",
		MaxAge:   60 * 60 * 24 * 7, // 7 วัน
		Path:     "/",
	})
}

// clearRefreshTokenCookie ลบ refresh_token cookie ออก (ใช้ตอน logout)
func clearRefreshTokenCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		MaxAge:   -1,
		Path:     "/",
	})
}

// What: รับ login request ของ user (เอา email + password + device info)
//
//	แล้วคืน access_token + refresh_token กลับ
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

	// What: ตั้ง refresh_token เป็น HttpOnly cookie (Approach B)
	// Why:  JS อ่านไม่ได้ → ป้องกัน XSS; browser ส่ง cookie อัตโนมัติเมื่อ refresh
	setRefreshTokenCookie(c, tokens.RefreshToken)
	return c.JSON(fiber.Map{"access_token": tokens.AccessToken})
}

// What: รับ login request ของ admin (เอา username + password)
//
//	แล้วคืน access_token + refresh_token กลับ
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

	setRefreshTokenCookie(c, tokens.RefreshToken)
	return c.JSON(fiber.Map{"access_token": tokens.AccessToken})
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

	// What: Fallback — อ่าน refresh_token จาก HttpOnly cookie ถ้าไม่ได้ส่งใน body
	if req.RefreshToken == "" {
		req.RefreshToken = c.Cookies("refresh_token")
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

	// What: ลบ cookie หลัง logout สำเร็จ
	clearRefreshTokenCookie(c)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

// What: รับ refresh_token แล้วออก access_token ใหม่
// Why:  รองรับการ login ของ client โดยไม่ต้องใส่ credentials ใหม่ทุกครั้ง
func (h *authHandler) RefreshToken(c *fiber.Ctx) error {
	// What: parse body สำหรับรับ refresh_token (optional — อาจมาจาก cookie)
	req := &dto.RefreshRequest{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	// What: Fallback — อ่าน refresh_token จาก HttpOnly cookie ถ้าไม่ได้ส่งใน body
	if req.RefreshToken == "" {
		req.RefreshToken = c.Cookies("refresh_token")
	}

	if req.RefreshToken == "" {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("refresh_token is required"))
	}

	tokens, err := h.authSvc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	// What: rotate refresh_token cookie ด้วย token ใหม่ (refresh token rotation)
	setRefreshTokenCookie(c, tokens.RefreshToken)
	return c.JSON(fiber.Map{"access_token": tokens.AccessToken})
}

// What: รับ Google ID token จาก frontend แล้ว verify + find-or-create user
// Why:  frontend ใช้ Google Identity Services เพื่อรับ credential (JWT) แล้วส่งมาที่นี่
//
//	backend verify กับ Google แล้วออก token ของระบบให้
func (h *authHandler) GoogleLogin(c *fiber.Ctx) error {
	var req struct {
		IDToken    string `json:"id_token"    validate:"required"`
		DeviceInfo string `json:"device_info"`
	}

	if err := c.BodyParser(&req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Invalid request body or format"))
	}

	if err := h.validator.Struct(&req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("Validation failed: "+err.Error()))
	}

	tokens, err := h.authSvc.GoogleLoginUser(c.Context(), req.IDToken, c.IP(), req.DeviceInfo)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	setRefreshTokenCookie(c, tokens.RefreshToken)
	return c.JSON(fiber.Map{"access_token": tokens.AccessToken})
}

// GetMe คืนโปรไฟล์ของผู้ใช้ปัจจุบัน โดยเช็ค role จาก JWT
// Why: GET /users/me ดึงจาก user table เท่านั้น — admin token จะหา user ไม่เจอ (404)
//
//	endpoint นี้เป็น unified /auth/me ที่ทำงานได้กับทั้ง user และ admin token
func (h *authHandler) GetMe(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("unauthorized"))
	}
	role, _ := c.Locals("role").(string)

	if role == "admin" {
		profile, err := h.adminSvc.GetProfile(c.Context(), userID)
		if err != nil {
			return httpcore.HandleError(c, err)
		}
		return c.JSON(profile)
	}

	profile, err := h.userSvc.GetProfile(c.Context(), userID)
	if err != nil {
		return httpcore.HandleError(c, err)
	}
	return c.JSON(profile)
}
