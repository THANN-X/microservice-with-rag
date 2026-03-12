package main

import (
	"authmiddleware"
	"context"
	"fmt"
	"jwtutils"
	"log"
	"os"
	"os/signal"
	"product_service/internal/adapter/handler/http"
	"product_service/internal/adapter/handler/message"
	"product_service/internal/adapter/messaging/consumer"
	"product_service/internal/adapter/messaging/producer"
	repository "product_service/internal/adapter/repository/postgres"
	"product_service/internal/config"
	"product_service/internal/core/service/command"
	"product_service/internal/core/service/query"
	"product_service/internal/core/service/worker"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"

	_ "product_service/docs"
)

// @title Product Service API
// @version 1.0
// @description This is the Product Service API using Hexagonal Architecture & CQRS.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email your.email@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer " followed by your token. Example: "Bearer eyJhbGci..."

// @host localhost:3002
// @BasePath /
func main() {
	// === SETUP DATABASE ===
	cfg := config.Loadconfig()
	dsn := cfg.GetDSN()
	db := config.OpenDatabase(dsn)

	// === DEPENDENCY INJECTION (Repositories) ===
	// WHY ใช้ DI pattern?
	//   - Service layer ไม่ depend ตรงๆ กับ GORM/Postgres → testable + swappable
	//   - แต่ละ repository คืน 2 interfaces (Command + Query) จาก struct เดียวกัน (CQRS with shared DB)
	cmdRepo, queryRepo := repository.NewProductRepository(db)
	inboxRepo := repository.NewInboxRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)
	catCmdRepo, catQueryRepo := repository.NewCategoryRepository(db)
	attrCmdRepo, attrQueryRepo := repository.NewAttributeRepository(db)

	// JWT Setup
	_ = godotenv.Load()
	value, _ := os.LookupEnv("JWT_SECRET")
	if value == "" {
		value = "my-secret-key-change-me" // Fallback (Dev only)
	}

	fmt.Println(value)

	// Middleware Instance
	jwtService := jwtutils.NewJWTService(value, "ecommerce_app")
	authMiddleware := authmiddleware.AuthMiddleware(jwtService)

	// === DEPENDENCY INJECTION (Services) ===
	cmdService := command.NewProductCommandService(cmdRepo, outboxRepo, inboxRepo)
	queryService := query.NewProductQueryService(queryRepo)
	catCmdService := command.NewCategoryCommandService(catCmdRepo)
	catQueryService := query.NewCategoryQueryService(catQueryRepo)
	attrCmdService := command.NewAttributeCommandService(attrCmdRepo, attrQueryRepo)
	attrQueryService := query.NewAttributeQueryService(attrQueryRepo)

	// Setup Kafka Producer
	/********************************************************************************/
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9094" // ค่า Default สำหรับรัน local (พอร์ต 9094 ตาม compose ใหม่)
	}
	fmt.Println(kafkaBrokers)

	KafkaProducer, err := producer.NewSaramaProducer([]string{kafkaBrokers})

	if err != nil {
		panic("Failed to create Kafka producer: " + err.Error())
	}
	defer KafkaProducer.Close()

	// Setup Background Workers (Outbox)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Outbox Processor in background
	outboxProcessor := worker.NewOutboxProcessor(outboxRepo, KafkaProducer)
	go outboxProcessor.Start(ctx)
	/********************************************************************************/

	//Setup Kafka Consumer
	/********************************************************************************/
	msgHandler := message.NewProductMessageHandler(cmdService)
	consumerGroupHandler := consumer.NewConsumerGroupHandler(msgHandler)

	// Config Consumer Group
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_1_0                      // หรือเวอร์ชันที่คุณใช้
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest // ถ้าเริ่มรันใหม่ให้อ่านตั้งแต่ต้น (หรือ OffsetNewest)
	saramaConfig.Consumer.Return.Errors = true

	// สร้าง Consumer Group (ตั้งชื่อ Group ให้ตรงกับ Service)
	consumerGroup, err := sarama.NewConsumerGroup([]string{kafkaBrokers}, "product-service-group", saramaConfig)
	if err != nil {
		panic("Failed to create Kafka consumer group: " + err.Error())
	}
	defer consumerGroup.Close()

	// สั่งรัน Consumer Group ใน Background
	go func() {
		// วนลูปเผื่อกรณีมีการ Rebalance หรือหลุด
		for {
			// ดักฟัง Topic ชื่อ "order.events" (สมมติว่าเป็น Topic จาก Order Service)
			if err := consumerGroup.Consume(ctx, []string{"order.events"}, consumerGroupHandler); err != nil {
				log.Printf("Error from consumer: %v\n", err)
			}
			// เช็คว่าถ้า context ถูกสั่ง cancel (ปิดเซิร์ฟเวอร์) ให้ออกจากลูป
			if ctx.Err() != nil {
				return
			}
		}
	}()
	/********************************************************************************/

	// Setup HTTP Handler & Routes
	app := fiber.New()

	// CORS Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOWED_ORIGINS"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Rate Limiter Middleware
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	// HTTP Handler
	handler := http.NewProductHandler(cmdService, queryService)
	catHandler := http.NewCategoryHandler(catCmdService, catQueryService)
	attrHandler := http.NewAttributeHandler(attrCmdService, attrQueryService)

	// Routes
	products := app.Group("/products")

	// Public / User Routes
	products.Get("/", handler.ListProducts)
	products.Get("/:id", handler.GetProduct)

	// Admin Only Routes
	adminGroup := products.Group("/admin", authMiddleware, authmiddleware.AdminGuard)

	// Create
	adminGroup.Post("/", handler.CreateProduct)
	adminGroup.Delete("/:id", handler.DeleteProduct)

	// Update General Info
	adminGroup.Put("/:id/general-info", handler.UpdateGeneralInfo)

	// Variant Management
	adminGroup.Post("/:id/variants", handler.AddVariant)
	adminGroup.Patch("/:id/variants/:variantId/price", handler.UpdateVariantPrice)
	adminGroup.Patch("/:id/variants/:variantId/stock", handler.AdjustStock)

	// Active / Inactive Management
	adminGroup.Patch("/:id/active", handler.SetProductActive)
	adminGroup.Patch("/:id/variants/:variantId/active", handler.SetVariantActive)

	// Category Routes
	categories := app.Group("/categories")
	categories.Get("/", catHandler.ListCategories)
	categories.Get("/:id", catHandler.GetCategory)

	catAdminGroup := categories.Group("/admin", authMiddleware, authmiddleware.AdminGuard)
	catAdminGroup.Post("/", catHandler.CreateCategory)
	catAdminGroup.Put("/:id", catHandler.UpdateCategory)
	catAdminGroup.Delete("/:id", catHandler.DeleteCategory)
	catAdminGroup.Patch("/:id/active", catHandler.SetCategoryActive)

	// Attribute Routes
	attributes := app.Group("/attributes")
	attributes.Get("/", attrHandler.ListAttributes)
	attributes.Get("/:id", attrHandler.GetAttribute)

	attrAdminGroup := attributes.Group("/admin", authMiddleware, authmiddleware.AdminGuard)
	attrAdminGroup.Post("/", attrHandler.CreateAttribute)
	attrAdminGroup.Put("/:id", attrHandler.UpdateAttribute)
	attrAdminGroup.Delete("/:id", attrHandler.DeleteAttribute)
	attrAdminGroup.Post("/:id/values", attrHandler.CreateAttributeValue)
	attrAdminGroup.Delete("/:id/values/:valueId", attrHandler.DeleteAttributeValue)

	// Graceful Shutdown (สำคัญสำหรับ Microservices!)
	// สร้าง Channel เพื่อรอรับสัญญาณปิดโปรแกรม (Ctrl+C หรือ Docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	app.Get("/swagger/*", swagger.HandlerDefault)

	// ให้เซิร์ฟเวอร์รันใน Goroutine เพื่อไม่ให้ block
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3002"
		}
		fmt.Printf("🚀 Product Service is running on port %s\n", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// รอรับสัญญาณปิดโปรแกรม
	<-quit
	fmt.Println("Gracefully shutting down server...")

	// แจ้ง Worker ให้หยุดทำงาน (ผ่าน Context Cancel)
	cancel()

	// รอให้ HTTP Server ปิดตัวเองอย่างปลอดภัย (ไม่ตัด request ที่กำลังทำงานอยู่)
	_ = app.Shutdown()

	fmt.Println("Server was successful shutdown.")

}
