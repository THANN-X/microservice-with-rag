package httpcore

import (
	"errs"

	"github.com/gofiber/fiber/v2"
)

// --- Centralized Error Helper ---
// ฟังก์ชันนี้จะ "แกะ" error ดูว่าเป็น AppError ของเราไหม แล้วตอบกลับด้วย Code ที่ Service กำหนดมา
func HandleError(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(errs.AppError); ok {
		return c.Status(appErr.Code).JSON(fiber.Map{
			"error": appErr.Message,
		})
	}
	// 2. (เพิ่ม) เช็คว่าเป็น Error จากการ Parse ของ Fiber โดยตรงหรือไม่
	// fiber.Error เป็น struct ของ framework เองที่มักจะเกิดตอน BodyParser หรือ Routing
	if fiberErr, ok := err.(*fiber.Error); ok {
		return c.Status(fiberErr.Code).JSON(fiber.Map{
			"error": fiberErr.Message,
		})
	}

	// กรณีเป็น error อื่นๆ ที่เราไม่ได้ Handle (เช่น Database down หลุดมา)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Internal Server Error",
	})
}

// ------------------------------
