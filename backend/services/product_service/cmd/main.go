package main

import (
	"authmiddleware"
	"context"
	"fmt"
	"health"
	"messaging"
	"os"
	"product_service/internal/adapter/handler/http"
	"product_service/internal/adapter/handler/message"
	"product_service/internal/adapter/messaging/producer"
	repository "product_service/internal/adapter/repository/postgres"
	"product_service/internal/config"
	"product_service/internal/core/service/command"
	"product_service/internal/core/service/query"
	"product_service/internal/core/service/worker"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
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

	_ = godotenv.Load()

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
	// DLQ: message ที่ retry ครบแล้วยังพัง → order.events.dlq แทนการทิ้ง
	dlq, err := messaging.NewDeadLetterPublisher([]string{kafkaBrokers})
	if err != nil {
		panic("Failed to create Kafka dead letter publisher: " + err.Error())
	}
	defer dlq.Close()

	msgHandler := message.NewProductMessageHandler(cmdService)
	consumerGroupHandler := messaging.NewConsumerGroupHandler("product", msgHandler, dlq, messaging.DefaultRetryPolicy())

	// Config Consumer Group
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = messaging.KafkaVersion               // ค่าเดียวกันทั้งระบบ (ดู pkg/messaging/kafka.go)
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest // ถ้าเริ่มรันใหม่ให้อ่านตั้งแต่ต้น (หรือ OffsetNewest)
	saramaConfig.Consumer.Return.Errors = true                  // อ่านโดย messaging.LogConsumerErrors ข้างล่าง

	// สร้าง Consumer Group (ตั้งชื่อ Group ให้ตรงกับ Service)
	consumerGroup, err := sarama.NewConsumerGroup([]string{kafkaBrokers}, "product-service-group", saramaConfig)
	if err != nil {
		panic("Failed to create Kafka consumer group: " + err.Error())
	}
	defer consumerGroup.Close()
	go messaging.LogConsumerErrors("product", consumerGroup)

	// สั่งรัน Consumer Group ใน Background ดักฟัง "order.events" จาก Order Service
	// RunConsumerGroup วนเข้ากลุ่มใหม่เองเมื่อ Rebalance, รอ backoff เมื่อ Kafka หลุด และออกเมื่อปิดเซิร์ฟเวอร์
	go messaging.RunConsumerGroup(ctx, "product", consumerGroup, []string{"order.events"}, consumerGroupHandler)
	/********************************************************************************/

	// Setup HTTP Handler & Routes
	app := fiber.New()

	// Health: ต้องลงทะเบียนก่อน route ที่มี InternalAuth
	sqlDB, err := db.DB()
	if err != nil {
		panic("Failed to get sql.DB from gorm: " + err.Error())
	}
	checker := health.New("product").Add("postgres", health.SQL(sqlDB))
	checker.Register(app)

	// HTTP Handler
	handler := http.NewProductHandler(cmdService, queryService)
	catHandler := http.NewCategoryHandler(catCmdService, catQueryService)
	attrHandler := http.NewAttributeHandler(attrCmdService, attrQueryService)

	// Routes
	products := app.Group("/products")

	// Public / User Routes
	products.Get("/", handler.ListProducts)
	products.Get("/:id", handler.GetProduct)

	// Admin Only Routes — BFF validate JWT + admin role แล้ว, service เช็คซ้ำ (defense-in-depth)
	internalAuth := authmiddleware.InternalAuthMiddleware()
	adminGroup := products.Group("/admin", internalAuth, authmiddleware.InternalAdminGuard)

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

	// Image Management
	adminGroup.Patch("/:id/images", handler.UpdateProductImages)
	adminGroup.Patch("/:id/variants/:variantId/images", handler.UpdateVariantImages)

	// Category Routes
	categories := app.Group("/categories")
	categories.Get("/", catHandler.ListCategories)
	categories.Get("/:id", catHandler.GetCategory)

	catAdminGroup := categories.Group("/admin", internalAuth, authmiddleware.InternalAdminGuard)
	catAdminGroup.Post("/", catHandler.CreateCategory)
	catAdminGroup.Put("/:id", catHandler.UpdateCategory)
	catAdminGroup.Delete("/:id", catHandler.DeleteCategory)
	catAdminGroup.Patch("/:id/active", catHandler.SetCategoryActive)

	// Attribute Routes
	attributes := app.Group("/attributes")
	attributes.Get("/", attrHandler.ListAttributes)
	attributes.Get("/:id", attrHandler.GetAttribute)

	attrAdminGroup := attributes.Group("/admin", internalAuth, authmiddleware.InternalAdminGuard)
	attrAdminGroup.Post("/", attrHandler.CreateAttribute)
	attrAdminGroup.Put("/:id", attrHandler.UpdateAttribute)
	attrAdminGroup.Delete("/:id", attrHandler.DeleteAttribute)
	attrAdminGroup.Post("/:id/values", attrHandler.CreateAttributeValue)
	attrAdminGroup.Delete("/:id/values/:valueId", attrHandler.DeleteAttributeValue)

	app.Get("/swagger/*", swagger.HandlerDefault)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3002"
	}
	fmt.Printf("🚀 Product Service is running on port %s\n", port)

	// Graceful Shutdown (สำคัญสำหรับ Microservices!)
	// Serve block จนได้ Ctrl+C / docker stop → /readyz ตอบ 503 → รอ request ที่ค้างอยู่ให้เสร็จ
	if err := checker.Serve(app, ":"+port); err != nil {
		// panic แทน log.Fatalf: panic ยังรัน defer (ปิด consumer group / DLQ / producer) ส่วน log.Fatalf เรียก os.Exit ข้ามทิ้งหมด
		panic("Server failed: " + err.Error())
	}

	// HTTP ปิดแล้ว → แจ้ง Worker (outbox + Kafka consumer) ให้หยุดทำงาน (ผ่าน Context Cancel)
	// WHY ทีหลัง HTTP? request ที่ค้างอยู่อาจเพิ่งเขียน outbox ต้องให้ worker ยังอยู่จนถึงตอนนั้น
	cancel()

	fmt.Println("Server was successful shutdown.")
}
