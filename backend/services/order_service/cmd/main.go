// WHAT: Entry point ของ order_service — DI wiring, routing, graceful shutdown
//
// Architecture Overview:
//   HTTP Request → Fiber Router → Handler → Service → Repository → DB
//   Kafka msg (stock.events) → ConsumerGroup → MessageHandler → Service → Repository
//   ODB event → OutboxProcessor → Kafka Producer → Kafka (order.events)
//
// Saga Participation:
//   Producer: order.events (OrderCreatedEvent, OrderCancelledEvent, OrderConfirmedEvent)
//   Consumer: stock.events (StockReservedEvent)
package main

import (
	"authmiddleware"
	"context"
	"health"
	"log"
	"os"
	"time"

	// handlers
	"order_service/internal/adapter/client"
	httphandler "order_service/internal/adapter/handler/http"
	msghandler "order_service/internal/adapter/handler/message"
	// messaging
	"order_service/internal/adapter/messaging/producer"
	// repository
	repository "order_service/internal/adapter/repository/postgres"
	// config
	"order_service/internal/config"
	// gateway port interface
	gatewayport "order_service/internal/core/port/gateway"
	// services
	"order_service/internal/core/service/command"
	"order_service/internal/core/service/query"
	woker "order_service/internal/core/service/woker"

	"messaging"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	// ─── Config + DB ──────────────────────────────────────────────────────────
	cfg := config.Loadconfig()
	db := config.OpenDatabase(cfg.GetDSN())

	// ─── Repositories ─────────────────────────────────────────────────────────
	cmdRepo, queryRepo := repository.NewOrderRepository(db)
	inboxRepo := repository.NewInboxRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	// Auto-select gateway: ถ้ามี STRIPE_SECRET_KEY → ใช้ Stripe, ถ้าไม่มี → ใช้ Stub (dev/test)
	var paymentGateway gatewayport.PaymentGateway
	if cfg.StripeSecretKey != "" {
		paymentGateway = client.NewStripeGateway(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
		log.Println("Payment gateway: Stripe (live/test mode)")
	} else {
		paymentGateway = client.NewStubPaymentGateway()
		log.Println("Payment gateway: Stub (set STRIPE_SECRET_KEY to enable Stripe)")
	}

	_ = godotenv.Load()

	// ─── Services ─────────────────────────────────────────────────────────────
	catalogClient := client.NewCatalogClient(cfg.CatalogServiceURL)
	cmdService := command.NewOrderCommandService(cmdRepo, inboxRepo, paymentRepo, paymentGateway, catalogClient)
	queryService := query.NewOrderQueryService(queryRepo)

	// ─── Kafka Producer ───────────────────────────────────────────────────────
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9094"
	}

	kafkaProducer, err := producer.NewSaramaProducer([]string{kafkaBrokers})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// ─── Context + Background Workers ─────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// OutboxProcessor: polls outbox table และ publish pending events ไป Kafka
	outboxProcessor := woker.NewOutboxProcessor(outboxRepo, kafkaProducer)
	go outboxProcessor.Start(ctx)

	// PaymentTimeoutChecker: auto-cancel orders ที่อยู่ใน AWAITING_PAYMENT นานเกิน 30 นาที
	paymentChecker := woker.NewPaymentTimeoutChecker(cmdRepo, paymentRepo, 30*time.Minute)
	go paymentChecker.Start(ctx)

	// ─── Kafka Consumer (stock.events) ────────────────────────────────────────
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = messaging.KafkaVersion
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Return.Errors = true // อ่านโดย messaging.LogConsumerErrors ข้างล่าง

	// DLQ: message ที่ retry ครบแล้วยังพัง → stock.events.dlq แทนการทิ้ง
	dlq, err := messaging.NewDeadLetterPublisher([]string{kafkaBrokers})
	if err != nil {
		log.Fatalf("Failed to create Kafka dead letter publisher: %v", err)
	}
	defer dlq.Close()

	msgHandler := msghandler.NewOrderMessageHandler(cmdService)
	consumerGroupHandler := messaging.NewConsumerGroupHandler("order", msgHandler, dlq, messaging.DefaultRetryPolicy())

	consumerGroup, err := sarama.NewConsumerGroup([]string{kafkaBrokers}, "order-service-group", saramaConfig)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer group: %v", err)
	}
	defer consumerGroup.Close()
	go messaging.LogConsumerErrors("order", consumerGroup)

	// WHY goroutine?
	//   - Consume blocks จน session end/rebalance และ RunConsumerGroup วนจน shutdown
	//   - ต้องรันควบคู่กับ HTTP server
	go messaging.RunConsumerGroup(ctx, "order", consumerGroup, []string{"stock.events"}, consumerGroupHandler)

	// ─── Fiber HTTP Server ────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		// WHY ErrorHandler? → centralize panic recovery + structured error response
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		},
	})

	// ─── Health ───────────────────────────────────────────────────────────────
	// WHY ก่อน Routes? → /healthz, /readyz ต้องไม่โดน InternalAuth
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB from gorm: %v", err)
	}
	checker := health.New("order").Add("postgres", health.SQL(sqlDB))
	checker.Register(app)

	// ─── Routes ───────────────────────────────────────────────────────────────
	handler := httphandler.NewOrderHandler(cmdService, queryService)

	// Customer routes — BFF validate JWT แล้ว ส่ง X-User-ID/X-Role มา
	internalAuth := authmiddleware.InternalAuthMiddleware()
	orders := app.Group("/orders", internalAuth)
	orders.Post("/", handler.PlaceOrder)
	orders.Get("/:id", handler.GetOrder)
	orders.Post("/:id/cancel", handler.CancelOrder)
	orders.Post("/:id/pay", handler.ProcessPayment)

	// Admin routes — BFF ตรวจสิทธิ์ admin แล้ว, service เช็คซ้ำ (defense-in-depth)
	adminOrders := app.Group("/orders/admin", internalAuth, authmiddleware.InternalAdminGuard)
	adminOrders.Post("/:id/cancel", handler.AdminCancelOrder)

	// Webhook route (ไม่มี auth middleware — payment gateway เรียกโดยตรง)
	app.Post("/webhook/payment", handler.HandlePaymentWebhook)

	// ─── Graceful Shutdown ────────────────────────────────────────────────────
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3003" // WHY 3003? auth=3001, product=3002, order=3003
	}

	// Block จนได้ signal shutdown → /readyz ตอบ 503 → รอ request ที่ค้างอยู่ (เช่น webhook ชำระเงิน) ให้เสร็จ
	if err := checker.Serve(app, ":"+port); err != nil {
		// panic แทน log.Fatalf: panic ยังรัน defer (ปิด consumer group / DLQ / producer) ส่วน log.Fatalf เรียก os.Exit ข้ามทิ้งหมด
		panic("Order service failed: " + err.Error())
	}

	// HTTP ปิดแล้ว → Cancel context → stop OutboxProcessor + Kafka Consumer
	// WHY ทีหลัง HTTP? request ที่ค้างอยู่อาจเพิ่งเขียน outbox ต้องให้ worker ยังอยู่จนถึงตอนนั้น
	cancel()

	log.Println("Order service stopped.")
}

