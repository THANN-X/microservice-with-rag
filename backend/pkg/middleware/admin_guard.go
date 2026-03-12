package authmiddleware

import (
	"errs"
	"httpcore"

	"github.com/gofiber/fiber/v2"
)

// AdminGuard ต้องวางต่อท้าย AuthMiddleware เท่านั้น
func AdminGuard(c *fiber.Ctx) error {
	// ดึง Role จาก Locals (ที่ AuthMiddleware แปะไว้ให้)
	role, ok := c.Locals("role").(string)
	if !ok {
		// ถ้าไม่มี role แสดงว่าอาจจะลืมใส่ AuthMiddleware ไว้ข้างหน้า
		return httpcore.HandleError(c, errs.NewUnauthorizedError("User role not found"))
	}

	// เช็คว่าเป็น Admin หรือไม่
	if role != "admin" {
		// 403 Forbidden: รู้จักนะว่าคุณคือใคร (User) แต่คุณไม่มีสิทธิ์เข้าห้องนี้
		return httpcore.HandleError(c, errs.NewForbiddenError("Access denied: Admins only"))
	}

	// ถ้าใช่ Admin ก็ให้ผ่านไปทำ Handler ต่อ
	return c.Next()
}
