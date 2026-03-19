package main

import (
	"auth_service/internal/adapter/handler/http"
	repository "auth_service/internal/adapter/repository/postgres"
	"authmiddleware"
	"fmt"
	"jwtutils"
	"logs"
	"os"

	"auth_service/internal/config"
	"auth_service/internal/core/service"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// What: Composition Root — จุดเดียวที่ wiring dependency ทั้งหมดเข้าด้วยกัน
// Why:  ทำให้ inner layers (domain, service) ไม่รู้จัก framework ใด ๆ
//       และง่ายต่อการ swap implementation ในอนาคต (เช่น เปลี่ยน DB หรือ HTTP framework)
func main() {
	// What: โหลด config จาก environment variable แล้วเปิด connection ไปยัง Postgres
	cfg := config.LoadConfig()
	dsn := cfg.GetDSN()
	db := config.OpenDatabase(dsn)

	// What: สร้าง Repository instances — ชั้นที่คุยกับ DB โดยตรง
	// Why:  แยก Repository ออกเป็น interface ทำให้ service layer test ได้โดยไม่ต้องใช้ DB จริง
	userRepository := repository.NewUserRepository(db)
	adminRepository := repository.NewAdminRepository(db)
	sessionRepository := repository.NewSessionRepository(db)

	// What: โหลด JWT_SECRET จาก env เพื่อใช้ sign/verify token
	_ = godotenv.Load()
	value, _ := os.LookupEnv("JWT_SECRET")
	if value == "" {
		// Why: fallback นี้มีไว้สำหรับ local dev เท่านั้น
		// TODO: ให้ panic หรือ log warning ถ้าไม่มีค่าใน production environment
		value = "my-secret-key-change-me"
	}

	// What: inline middleware สำหรับป้องกัน endpoint สร้าง Admin
	// Why:  Admin ไม่ควรสมัครเองได้ — ต้องรู้ secret key ของระบบก่อน
	// TODO: ย้าย adminSecretGuard ออกไปอยู่ใน pkg/middleware เพื่อ reuse ข้าม service
	adminSecretGuard := func(c *fiber.Ctx) error {
		expectedSecret := os.Getenv("ADMIN_SECRET_KEY")
		if expectedSecret == "" {
			// Why: fallback สำหรับ local dev เท่านั้น ห้ามใช้ใน production
			expectedSecret = "super-secret-admin-key"
		}

		// What: ดึง secret จาก request header แล้วเปรียบเทียบกับค่าที่ระบบกำหนด
		clientSecret := c.Get("X-Admin-Secret")
		if clientSecret != expectedSecret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Invalid admin secret key",
			})
		}
		return c.Next()
	}

	fmt.Println(value)

	// What: JWTService ใช้สำหรับ generate และ validate token ทั้ง access และ refresh
	jwtService := jwtutils.NewJWTService(value, "ecommerce_app")

	// What: สร้าง Service instances — ชั้น Business Logic
	userService := service.NewUserService(userRepository)
	adminService := service.NewAdminService(adminRepository)
	// Why: authService ต้องรู้จักทั้ง userRepo, adminRepo, sessionRepo
	//      เพราะ login ได้สองแบบ (user / admin) และต้องจัดการ session เอง
	authservice := service.NewAuthService(userRepository, adminRepository, sessionRepository, jwtService)

	// What: สร้าง Handler instances — ชั้น HTTP Adapter รับ/ส่ง request
	userHandler := http.NewUserHandler(userService)
	adminHandler := http.NewAdminHandler(adminService)
	authHandler := http.NewAuthHandler(userService, adminService, authservice)

	// What: สร้าง JWT middleware สำหรับตรวจสอบ access token ใน header
	authMiddleware := authmiddleware.AuthMiddleware(jwtService)

	// What: สร้าง Fiber app instance
	app := fiber.New()

	// --- Public Routes (ไม่ต้อง login) ---

	// What: endpoint สมัครสมาชิก user ทั่วไป ไม่ต้องมี token
	app.Post("/users/register", userHandler.RegisterUser)

	// What: endpoint สร้าง admin ต้องมี ADMIN_SECRET_KEY ใน header ก่อน
	app.Post("/admin/register", adminSecretGuard, adminHandler.RegisterAdmin)

	// --- Auth Routes ---
	authGroup := app.Group("/auth")
	// What: login แล้วรับ access_token + refresh_token กลับมา
	authGroup.Post("/user-login", authHandler.LoginUser)
	authGroup.Post("/admin-login", authHandler.LoginAdmin)
	// What: revoke refresh token ทำให้ session หมดอายุทันที
	authGroup.Post("/logout", authHandler.Logout)
	// What: แลก refresh_token เก่า → ได้ access_token ใหม่
	authGroup.Post("/refresh-token", authHandler.RefreshToken)

	// --- Protected Routes (ต้องมี valid access_token) ---
	// Why: ใช้ Group + Middleware เพื่อให้ทุก route ในกลุ่มถูก intercept โดย authMiddleware
	protected := app.Group("/users", authMiddleware)

	protected.Post("/update/:id", userHandler.UpdateProfile)
	protected.Post("/chgpass/:id", userHandler.ChangePassword)
	// Why: ต้องประกาศ /me ก่อน /:id เสมอ — Fiber จับ route แบบ first-match
	//      ถ้าประกาศ /:id ก่อน คำว่า "me" จะถูกดักเป็น param แทน
	protected.Get("/me", userHandler.GetMyProfile)
	// What: ดึงโปรไฟล์ของ user ตาม ID (admin หรือเจ้าของเท่านั้น)
	protected.Get("/:id", userHandler.GetProfile)

	logs.Info("Auth service started at port 3001")

	// What: เริ่ม listen HTTP server ที่ port 3001
	// TODO: รองรับ graceful shutdown (trap SIGINT/SIGTERM แล้ว app.ShutdownWithTimeout)
	app.Listen(":3001")
}
