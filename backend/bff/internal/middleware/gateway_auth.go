package middleware

import (
	"jwtutils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GatewayAuth ทำหน้าที่เป็น centralized auth สำหรับทุก request ที่ผ่าน BFF
//   - Public routes: ปล่อยผ่านเลย
//   - Protected routes: ต้องมี valid JWT (access token)
//   - Admin routes: ต้องมี valid JWT + role=admin
func GatewayAuth(jwtService *jwtutils.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		method := c.Method()

		// ─── Public routes → ไม่ต้องมี Token ───
		if isPublicRoute(path, method) {
			return c.Next()
		}

		// ─── ดึง Authorization header ───
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token format",
			})
		}

		// ─── Validate JWT ───
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		if claims.Type != jwtutils.AccessToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token type",
			})
		}

		// ─── Admin routes → ต้อง role=admin ───
		if isAdminRoute(path) && claims.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied: Admins only",
			})
		}

		// ─── ฝัง user info ใน Locals สำหรับ proxy forward ───
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// isPublicRoute ตรวจสอบว่า route นี้เป็น public หรือไม่
func isPublicRoute(path, method string) bool {
	// Auth: login, logout, refresh-token, google — เป็น public เพราะยังไม่มี token
	// Why: /api/auth/auth/me ต้องการ JWT — ยกเว้นออกจาก public rule
	//      ถ้าปล่อยผ่านโดยไม่ validate JWT → BFF ไม่ set X-User-ID/X-Role headers
	//      → InternalAuthMiddleware ใน auth-service ไม่เจอ X-User-ID → 401 ทุกครั้ง
	if strings.HasPrefix(path, "/api/auth/auth/") && path != "/api/auth/auth/me" {
		return true
	}
	// Auth: user register
	if path == "/api/auth/users/register" && method == "POST" {
		return true
	}
	// Auth: admin register (ใช้ X-Admin-Secret แยก — ไม่ใช่ JWT)
	if path == "/api/auth/admin/register" && method == "POST" {
		return true
	}

	// AI Chat
	if path == "/chat" {
		return true
	}

	// Catalog: ทุก route เป็น public (read-only search)
	if strings.HasPrefix(path, "/api/catalog") {
		return true
	}

	// Products, Categories, Attributes: GET ที่ไม่ใช่ admin path
	if method == "GET" {
		if strings.HasPrefix(path, "/api/products") && !strings.Contains(path, "/admin") {
			return true
		}
		if strings.HasPrefix(path, "/api/categories") && !strings.Contains(path, "/admin") {
			return true
		}
		if strings.HasPrefix(path, "/api/attributes") && !strings.Contains(path, "/admin") {
			return true
		}
	}

	return false
}

// isAdminRoute ตรวจสอบว่า route นี้ต้องการสิทธิ์ admin หรือไม่
func isAdminRoute(path string) bool {
	adminPrefixes := []string{
		"/api/products/admin",
		"/api/categories/admin",
		"/api/attributes/admin",
		"/api/orders/admin",
		"/api/order-history/admin",
	}
	for _, prefix := range adminPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
