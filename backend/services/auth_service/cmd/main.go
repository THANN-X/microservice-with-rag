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

func main() {
	// Setup Database
	cfg := config.LoadConfig()
	dsn := cfg.GetDSN()
	db := config.OpenDatabase(dsn)

	// Setup Repositories
	userRepository := repository.NewUserRepositoryDB(db)
	adminRepository := repository.NewAdminRepository(db)
	sessionRepository := repository.NewSessionRepository(db)

	// JWT Setup
	_ = godotenv.Load()
	value, _ := os.LookupEnv("JWT_SECRET")
	if value == "" {
		value = "my-secret-key-change-me" // Fallback (Dev only)
	}

	adminSecretGuard := func(c *fiber.Ctx) error {
		expectedSecret := os.Getenv("ADMIN_SECRET_KEY")
		if expectedSecret == "" {
			expectedSecret = "super-secret-admin-key" // Fallback (Dev only)
		}

		clientSecret := c.Get("X-Admin-Secret")
		if clientSecret != expectedSecret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Invalid admin secret key",
			})
		}
		return c.Next()
	}

	fmt.Println(value)

	jwtService := jwtutils.NewJWTService(value, "ecommerce_app")

	// Setup Services
	userService := service.NewUserService(userRepository)
	adminService := service.NewAdminService(adminRepository)
	authservice := service.NewAuthService(userRepository, adminRepository, sessionRepository, jwtService)

	// Setup Handlers
	userHandler := http.NewUserHandler(userService)
	adminHandler := http.NewAdminHandler(adminService)
	authHandler := http.NewAuthHandler(userService, adminService, authservice)

	// Middleware Instance
	authMiddleware := authmiddleware.AuthMiddleware(jwtService)

	// Setup HTTP Handler & Routes
	app := fiber.New()

	// Public Routes
	app.Post("/users/register", userHandler.RegisterUser)

	// Protected Route ด้วย Secret Key
	app.Post("/admin/register", adminSecretGuard, adminHandler.CreateAdmin)

	// Auth Routes
	authGroup := app.Group("/auth")
	authGroup.Post("/user-login", authHandler.LoginUser)
	authGroup.Post("/admin-login", authHandler.LoginAdmin)
	authGroup.Post("/logout", authHandler.Logout)
	authGroup.Post("/refresh-token", authHandler.RefreshToken)

	// Protected Routes
	// กลุ่มนี้จะถูกดักด้วย Middleware ก่อนเสมอ
	protected := app.Group("/users", authMiddleware)

	// User Profile Routes
	protected.Post("/update/:id", userHandler.UpdateUserProfile)
	protected.Post("/chgpass/:id", userHandler.ChangePassword)
	// ต้องประกาศเส้นทางนี้ก่อน เพื่อป้องกันการชนกับ dynamic param
	protected.Get("/me", userHandler.GetMyProfile)
	// ดึงข้อมูลโปรไฟล์ของตัวเอง
	protected.Get("/:id", userHandler.GetUserProfile)

	logs.Info("Auth service started at port 3001")

	// Start the server
	app.Listen(":3001")
}
