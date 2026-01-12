package authmiddleware

import (
	"jwtutils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware รับ JWTService เข้ามาเพื่อใช้ตรวจสอบ Token
func AuthMiddleware(jwtService *jwtutils.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. ดึง Header "Authorization"
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization header"})
		}

		// 2. ตัดคำว่า "Bearer " ออก
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // กรณีไม่มี Bearer
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}

		// 3. Validate Token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// 4. (สำคัญ) เช็คว่าเป็น Access Token เท่านั้น (ห้ามเอา Refresh Token มายิง API)
		if claims.Type != jwtutils.AccessToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token type"})
		}

		// 5. ผ่านฉลุย! ฝัง UserID ลงใน Context ให้ Handler ตัวถัดไปใช้ต่อ
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}
