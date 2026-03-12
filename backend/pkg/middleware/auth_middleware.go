package authmiddleware

import (
	"errs"
	"httpcore"
	"jwtutils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware รับ JWTService เข้ามาเพื่อใช้ตรวจสอบ Token
func AuthMiddleware(jwtService *jwtutils.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง Header "Authorization"
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return httpcore.HandleError(c, errs.NewUnauthorizedError("Missing authorization header"))
		}

		// ตัดคำว่า "Bearer " ออก
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // กรณีไม่มี Bearer
			return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid token format"))
		}

		// Validate Token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid or expired token"))
		}

		// (สำคัญ) เช็คว่าเป็น Access Token เท่านั้น (ห้ามเอา Refresh Token มายิง API)
		if claims.Type != jwtutils.AccessToken {
			return httpcore.HandleError(c, errs.NewUnauthorizedError("Invalid token type"))
		}

		// ฝังข้อมูลลง Locals (เพื่อให้ Middleware ตัวถัดไปใช้ต่อ)
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}
