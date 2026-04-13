package main

import (
	"authmiddleware"
	carthttp "cart_service/internal/adapter/handler/http"
	repository "cart_service/internal/adapter/repository/postgres"
	"cart_service/internal/config"
	"cart_service/internal/core/service/command"
	"cart_service/internal/core/service/query"
	"jwtutils"
	"logs"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/joho/godotenv"
)

func main() {
	cfg := config.Loadconfig()
	dsn := cfg.GetDSN()
	db := config.OpenDatabase(dsn)

	cmdRepo, queryRepo := repository.NewCartRepository(db)

	_ = godotenv.Load()
	jwtSecret, _ := os.LookupEnv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "my-secret-key-change-me"
	}

	jwtService := jwtutils.NewJWTService(jwtSecret, "ecommerce_app")
	authMiddleware := authmiddleware.AuthMiddleware(jwtService)

	cmdService := command.NewCartCommandService(cmdRepo)
	queryService := query.NewCartQueryService(queryRepo)

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOWED_ORIGINS"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	handler := carthttp.NewCartHandler(cmdService, queryService)

	// ทุก route ต้องผ่าน JWT Auth — userID ดึงจาก Token เสมอ
	cart := app.Group("/cart", authMiddleware)
	cart.Get("/", handler.GetCart)
	cart.Post("/items", handler.AddItem)
	cart.Put("/items/:variantId", handler.UpdateItemQuantity)
	cart.Delete("/items/:variantId", handler.RemoveItem)
	cart.Delete("/", handler.ClearCart)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3004"
	}

	logs.Info("Cart Service starting on port " + port)

	if err := app.Listen(":" + port); err != nil {
		panic("failed to start server: " + err.Error())
	}
}
