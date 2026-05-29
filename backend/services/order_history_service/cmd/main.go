package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	orderhistoryhttp "order_history_service/internal/adapter/handler/http"
	"order_history_service/internal/adapter/messaging/consumer"
	msghandler "order_history_service/internal/adapter/messaging/handler"
	mongorepo "order_history_service/internal/adapter/repository/mongo"
	"order_history_service/internal/config"
	"order_history_service/internal/core/service/command"
	"order_history_service/internal/core/service/query"

	authmiddleware "authmiddleware"

	"logs"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

const (
	kafkaTopic    = "order.events"
	consumerGroup = "order-history-service-group"
	defaultPort   = "3006"
)

func main() {
	_ = godotenv.Load()

	cfg := config.LoadConfig()

	// --- MongoDB ---
	mongoClient, err := config.ConnectMongo(cfg.MongoURI)
	if err != nil {
		panic("failed to connect to MongoDB: " + err.Error())
	}
	defer mongoClient.Disconnect(context.Background()) //nolint:errcheck

	db := mongoClient.Database(cfg.MongoDB)

	if err := mongorepo.EnsureIndexes(db); err != nil {
		panic("failed to ensure MongoDB indexes: " + err.Error())
	}

	writeRepo, readRepo := mongorepo.NewOrderHistoryRepository(db)
	inboxRepo := mongorepo.NewInboxRepository(db)

	cmdService := command.NewOrderHistoryCommandService(writeRepo, inboxRepo)
	queryService := query.NewOrderHistoryQueryService(readRepo)

	// --- Kafka consumer ---
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9094"
	}

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	kafkaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGrp, err := sarama.NewConsumerGroup([]string{brokers}, consumerGroup, kafkaCfg)
	if err != nil {
		panic("failed to create kafka consumer group: " + err.Error())
	}
	defer consumerGrp.Close() //nolint:errcheck

	eventHandler := msghandler.NewOrderEventHandler(cmdService)
	groupHandler := consumer.NewConsumerGroupHandler(eventHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka consume loop — restart on rebalance
	go func() {
		for {
			if err := consumerGrp.Consume(ctx, []string{kafkaTopic}, groupHandler); err != nil {
				logs.Error("order-history: kafka consume error: " + err.Error())
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// --- HTTP server ---
	app := fiber.New()

	handler := orderhistoryhttp.NewOrderHistoryHandler(queryService)

	// order-history routes — BFF validate JWT แล้ว ส่ง X-User-ID/X-Role มา
	history := app.Group("/order-history", authmiddleware.InternalAuthMiddleware())
	history.Get("/admin/stats", handler.GetAdminStats)         // ต้องอยู่ก่อน /admin/:orderId เพื่อไม่ถูก capture
	history.Get("/admin", handler.ListAllOrders)               // ต้องอยู่ก่อน /:orderId เพื่อไม่ให้ถูก capture
	history.Get("/admin/:orderId", handler.GetAdminOrder)      // admin get single order (ไม่ check ownership)
	history.Get("/", handler.ListMyOrders)
	history.Get("/:orderId", handler.GetOrder)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = defaultPort
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logs.Info("Order History Service starting on port " + port)
		if err := app.Listen(":" + port); err != nil {
			logs.Error(err)
		}
	}()

	<-quit
	logs.Info("Order History Service shutting down...")
	cancel()
	if err := app.Shutdown(); err != nil {
		logs.Error(err)
	}
}
