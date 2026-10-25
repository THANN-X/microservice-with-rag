package main

import (
	"context"
	"os"

	orderhistoryhttp "order_history_service/internal/adapter/handler/http"
	msghandler "order_history_service/internal/adapter/messaging/handler"
	mongorepo "order_history_service/internal/adapter/repository/mongo"
	"order_history_service/internal/config"
	"order_history_service/internal/core/service/command"
	"order_history_service/internal/core/service/query"

	authmiddleware "authmiddleware"

	"health"
	"logs"
	"messaging"

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
	kafkaCfg.Version = messaging.KafkaVersion
	kafkaCfg.Consumer.Return.Errors = true // อ่านโดย messaging.LogConsumerErrors ข้างล่าง
	kafkaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	kafkaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGrp, err := sarama.NewConsumerGroup([]string{brokers}, consumerGroup, kafkaCfg)
	if err != nil {
		panic("failed to create kafka consumer group: " + err.Error())
	}
	defer consumerGrp.Close() //nolint:errcheck
	go messaging.LogConsumerErrors("order-history", consumerGrp)

	// DLQ: message ที่ retry ครบแล้วยังพัง → order.events.dlq แทนการทิ้ง
	dlq, err := messaging.NewDeadLetterPublisher([]string{brokers})
	if err != nil {
		panic("failed to create kafka dead letter publisher: " + err.Error())
	}
	defer dlq.Close() //nolint:errcheck

	eventHandler := msghandler.NewOrderEventHandler(cmdService)
	groupHandler := messaging.NewConsumerGroupHandler("order-history", eventHandler, dlq, messaging.DefaultRetryPolicy())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka consume loop — เข้ากลุ่มใหม่หลัง rebalance, backoff ตอน Kafka ล่ม, ออกเมื่อ shutdown
	go messaging.RunConsumerGroup(ctx, "order-history", consumerGrp, []string{kafkaTopic}, groupHandler)

	// --- HTTP server ---
	app := fiber.New()

	checker := health.New("order-history").Add("mongo", func(ctx context.Context) error {
		return mongoClient.Ping(ctx, nil)
	})
	checker.Register(app)

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

	logs.Info("Order History Service starting on port " + port)
	err = checker.Serve(app, ":"+port)

	// HTTP ปิดแล้ว → หยุด Kafka consumer ต่อ แล้ว defer ปิด consumer group / DLQ / Mongo
	cancel()
	if err != nil {
		panic(err.Error()) // panic ยังรัน defer ก่อน exit ต่างจาก os.Exit
	}
}
