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
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	// handlers
	"order_service/internal/adapter/client"
	httphandler "order_service/internal/adapter/handler/http"
	msghandler "order_service/internal/adapter/handler/message"
	// messaging
	"order_service/internal/adapter/messaging/consumer"
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
	saramaConfig.Version = sarama.V2_8_1_0
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Return.Errors = true

	msgHandler := msghandler.NewOrderMessageHandler(cmdService)
	consumerGroupHandler := consumer.NewConsumerGroupHandler(msgHandler)

	consumerGroup, err := sarama.NewConsumerGroup([]string{kafkaBrokers}, "order-service-group", saramaConfig)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer group: %v", err)
	}
	defer consumerGroup.Close()

	// WHY goroutine?
	//   - consumerGroup.Consume blocks จน session end/rebalance
	//   - ต้องรันควบคู่กับ HTTP server
	go func() {
		for {
			if err := consumerGroup.Consume(ctx, []string{"stock.events"}, consumerGroupHandler); err != nil {
				log.Printf("Error from consumer: %v\n", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// ─── Fiber HTTP Server ────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		// WHY ErrorHandler? → centralize panic recovery + structured error response
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		},
	})

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
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := os.Getenv("APP_PORT")
		if port == "" {
			port = "3003" // WHY 3003? auth=3001, product=3002, order=3003
		}
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Order service failed to start: %v", err)
		}
	}()

	// Block จนได้ signal shutdown
	<-quit
	log.Println("Shutting down order service...")

	// Cancel context → stop OutboxProcessor + Kafka Consumer
	cancel()

	// Graceful HTTP shutdown
	if err := app.Shutdown(); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	log.Println("Order service stopped.")
}

