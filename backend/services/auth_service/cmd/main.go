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
	// Load database configuration from environment variables
	// Connect to the database
	cfg := config.LoadConfig()
	dsn := cfg.GetDSN()
	db := config.OpenDatabase(dsn)
	// Open database connection

	// --- Repositories ---
	// Initialize repository, service, and handler
	userRepository := repository.NewUserRepositoryDB(db)
	sessionRepository := repository.NewSessionRepository(db)

	// --- Shared Kernel (JWT) ---
	// *สำคัญ* ควรดึง Secret จาก Config/Env ไม่ควร Hardcode
	_ = godotenv.Load()
	value, _ := os.LookupEnv("JWT_SECRET")
	if value == "" {
		value = "my-secret-key-change-me" // Fallback (Dev only)
	}

	fmt.Println(value)

	jwtService := jwtutils.NewJWTService(value, "ecommerce_app")

	// --- Services ---
	userService := service.NewUserService(userRepository)
	authservice := service.NewAuthService(userRepository, sessionRepository, jwtService)

	// --- Handlers ---
	userHandler := http.NewUserHandler(userService)
	authHandler := http.NewAuthHandler(userService, authservice)

	// ctx := context.Background()

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel() // สำคัญ: ต้อง cancel เสมอเมื่อจบการทำงาน

	// newUser := &domain.User{
	// 	FirstName: "Thann",
	// 	LastName:  "Khom",
	// 	Email:     "thann2@example.com",
	// 	Password:  "securepassword",
	// 	Phone:     "123-456-7890",
	// 	Address:   "123 Main St, City, Country",
	// 	Role:      "",
	// }
	// users, err := userRepository.AllUsers(ctx)

	//users, err := userRepository.FindByEmail(ctx, "thann@example.com")

	// user, err := userRepository.FindById(ctx, 1)

	// update := map[string]interface{}{
	// 	"first_name": "UpdatedName",
	// 	"last_name":  "UpdatedLastName",
	// 	"role":       "admin",
	// }

	// err = userRepository.Save(ctx, newUser)
	// err = userRepository.Update(ctx, 1, update)
	// err = userRepository.Delete(ctx, 2)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for _, u := range users {
	// 	fmt.Printf("User: %+v\n", u.FirstName)
	// }
	//fmt.Println("User saved successfully:", user)
	// fmt.Println("Updated user:", user)
	// fmt.Println("User deleted successfully")

	// Initialize Fiber app and routes
	// สร้าง Middleware Instance
	authMiddleware := authmiddleware.AuthMiddleware(jwtService)

	app := fiber.New()

	// ===========================
	// 🟢 Public Routes (ไม่ต้อง Login)
	// ===========================
	authGroup := app.Group("/auth")
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh-token", authHandler.RefreshToken)

	// Register ปกติไม่ต้อง Login ก็สมัครได้
	app.Post("/users/register", userHandler.RegisterUser)

	// ===========================
	// 🔒 Private Routes (ต้อง Login)
	// ===========================
	// สร้าง Group ใหม่สำหรับ route ที่ต้องป้องกัน
	protected := app.Group("/users")

	// *** พระเอกอยู่ตรงนี้: สั่งให้ Group นี้ใช้ Middleware ***
	protected.Use(authMiddleware)

	// Route พวกนี้จะถูกดักด้วย Middleware ก่อนเสมอ
	protected.Post("/update/:id", userHandler.UpdateUserProfile)
	protected.Post("/chgpass/:id", userHandler.ChangePassword)
	// ✅ วาง /me ไว้ก่อนเสมอ
	protected.Get("/me", userHandler.GetMyProfile)
	// ⬇️ แล้วค่อยตามด้วย dynamic param
	protected.Get("/:id", userHandler.GetUserProfile) // ดึง Profile ตัวเอง

	logs.Info("Auth service started at port 3001")

	// Start the server
	app.Listen(":3001")
}
