package authmiddleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// InternalAuthMiddleware อ่าน X-User-ID และ X-Role headers ที่ BFF ส่งมา
// หลังจากที่ BFF validate JWT แล้ว → set ลง Locals เพื่อให้ handler ใช้ต่อได้เหมือนเดิม
//
// Why: เมื่อย้าย JWT validation ไปอยู่ที่ BFF แล้ว แต่ละ service ไม่ต้อง validate JWT เอง
//
//	แค่อ่าน user identity จาก trusted headers ที่ BFF ส่งมา
func InternalAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-User-ID")
		if userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing user identity",
			})
		}

		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid user identity",
			})
		}

		role := c.Get("X-Role")

		c.Locals("user_id", uint(userID))
		c.Locals("role", role)

		return c.Next()
	}
}

// InternalAdminGuard ต้องวางต่อท้าย InternalAuthMiddleware เท่านั้น
// Defense-in-Depth: BFF ตรวจ admin role แล้ว แต่ service เช็คซ้ำอีกรอบเพื่อความปลอดภัย
func InternalAdminGuard(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: Admins only",
		})
	}
	return c.Next()
}
