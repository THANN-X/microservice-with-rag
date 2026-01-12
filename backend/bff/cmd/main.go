package main

import (
	"bff/internal/adapter/client"
	"bff/internal/adapter/handler"
	"bff/internal/core/service"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
)

// Main function to start the BFF server
func main() {
	// Implementation to initialize and start the BFF server
	userServiceURL := os.Getenv("USER_SERVICE_URL")

	if userServiceURL == "" {
		userServiceURL = "http://localhost:3001"
	}

	fmt.Println(userServiceURL)

	userClient := client.NewUserHTTPClient(userServiceURL)
	// Initialize services
	profileService := service.NewProfileService(userClient)

	// Initialize handlers
	bffHandler := handler.NewBFFHandler(profileService)

	// Set up Fiber app and routes
	app := fiber.New()

	app.Get("/users/:id", bffHandler.GetUserProfile)

	// Start the server
	app.Listen(":8080")
}
