package handler

import (
	"bff/internal/adapter/client"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// What: ProxyHandler เป็น HTTP handler ที่ forward request ไปยัง backend service
//
// Why:  BFF ใช้ pattern นี้เพื่อ route request ไปยัง microservice ที่ถูกต้อง
//   - Frontend เรียก /api/auth/users/me → BFF strip prefix "/api/auth" → forward /users/me ไป auth-service
//   - Frontend เรียก /api/products/123  → BFF strip prefix "/api" → forward /products/123 ไป product-service
//   - ทุก service ใช้ ProxyHandler เดียวกัน แค่เปลี่ยน proxy client กับ prefix
//
// ความแตกต่างจาก ChatHandler:
//   - ProxyHandler = transparent proxy (ส่งต่อ request/response ตรงๆ ไม่แปลง)
//   - ChatHandler = protocol translation (แปลง REST → gRPC → REST กลับ)
//
// TODO: เพิ่ม rate limiting per-service เพื่อป้องกัน backend service โดน overload
type ProxyHandler struct {
	proxy       *client.ServiceProxy
	stripPrefix string
}

// What: สร้าง ProxyHandler ใหม่ โดยกำหนด proxy client และ prefix ที่จะตัดออก
// Why:  URL ที่ frontend เรียก (เช่น /api/auth/xxx) กับ URL ที่ backend ใช้ (เช่น /xxx) ต่างกัน
//
//	ต้อง strip prefix ออกก่อน forward เพื่อให้ backend รับ path ที่ถูกต้อง
//
// ตัวอย่าง:
//
//	NewProxyHandler(authProxy, "/api/auth")
//	→ /api/auth/users/me → strip "/api/auth" → forward "/users/me"
//
//	NewProxyHandler(productProxy, "/api")
//	→ /api/products/123 → strip "/api" → forward "/products/123"
func NewProxyHandler(proxy *client.ServiceProxy, stripPrefix string) *ProxyHandler {
	return &ProxyHandler{proxy: proxy, stripPrefix: stripPrefix}
}

// What: Handle เป็น Fiber handler ที่ strip prefix แล้ว forward request ไปยัง backend
// Why:  ใช้เป็น catch-all handler สำหรับ route group
//
//	เช่น app.All("/api/auth/*", handler.Handle) จะ match ทุก path ที่ขึ้นต้นด้วย /api/auth/
//
// Flow:
//  1. Fiber match route → เรียก Handle
//  2. Handle ดึง c.Path() (เช่น "/api/auth/users/me")
//  3. ตัด prefix ออก → ได้ targetPath (เช่น "/users/me")
//  4. ส่งต่อให้ ServiceProxy.Forward() → ยิง HTTP request ไป backend
func (h *ProxyHandler) Handle(c *fiber.Ctx) error {
	// What: ตัด prefix ออกจาก original path เพื่อได้ path ที่ backend service ใช้จริง
	// ตัวอย่าง:
	//   "/api/auth/users/me"     → strip "/api/auth" → "/users/me"
	//   "/api/products"          → strip "/api"      → "/products"
	//   "/api/cart/items/5"      → strip "/api"      → "/cart/items/5"
	originalPath := c.Path()
	targetPath := strings.TrimPrefix(originalPath, h.stripPrefix)
	if targetPath == "" {
		targetPath = "/"
	}
	return h.proxy.Forward(c, targetPath)
}
