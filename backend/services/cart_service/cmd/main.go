package main

import (
	"authmiddleware"
	carthttp "cart_service/internal/adapter/handler/http"
	repository "cart_service/internal/adapter/repository/postgres"
	"cart_service/internal/config"
	"cart_service/internal/core/service/command"
	"cart_service/internal/core/service/query"
	"logs"
	"os"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.Loadconfig()
	dsn := cfg.GetDSN()
	db := config.OpenDatabase(dsn)

	cmdRepo, queryRepo := repository.NewCartRepository(db)

	cmdService := command.NewCartCommandService(cmdRepo)
	queryService := query.NewCartQueryService(queryRepo)

	app := fiber.New()

	handler := carthttp.NewCartHandler(cmdService, queryService)

	// ทุก route ต้องผ่าน InternalAuth — BFF validate JWT แล้ว ส่ง X-User-ID/X-Role มา
	cart := app.Group("/cart", authmiddleware.InternalAuthMiddleware())
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
