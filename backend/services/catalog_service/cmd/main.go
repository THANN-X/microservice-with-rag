package main

import (
	cataloghttp "catalog_service/internal/adapter/handler/http"
	msghandler "catalog_service/internal/adapter/messaging/handler"
	mongorepo "catalog_service/internal/adapter/repository/mongo"
	"catalog_service/internal/config"
	"catalog_service/internal/core/service/command"
	"catalog_service/internal/core/service/query"
	"context"
	"health"
	"os"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"logs"
	"messaging"
)

const (
	kafkaTopic    = "product.events"
	consumerGroup = "catalog-service-group"
	defaultPort   = "3005"
)

func main() {
	_ = godotenv.Load()

	cfg := config.LoadConfig()

	mongoClient, err := config.ConnectMongo(cfg.MongoURI)
	if err != nil {
		panic("failed to connect to MongoDB: " + err.Error())
	}
	defer mongoClient.Disconnect(context.Background()) //nolint:errcheck

	db := mongoClient.Database(cfg.MongoDB)

	if err := mongorepo.EnsureIndexes(db); err != nil {
		panic("failed to ensure MongoDB indexes: " + err.Error())
	}

	writeRepo, readRepo := mongorepo.NewCatalogRepository(db)
	inboxRepo := mongorepo.NewInboxRepository(db)

	cmdService := command.NewCatalogCommandService(writeRepo, inboxRepo)
	queryService := query.NewCatalogQueryService(readRepo)

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
	// OffsetOldest: เริ่มอ่าน backlog ตั้งแต่ต้นถ้า consumer group ยังไม่เคย commit offset
	kafkaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGrp, err := sarama.NewConsumerGroup([]string{brokers}, consumerGroup, kafkaCfg)
	if err != nil {
		panic("failed to create kafka consumer group: " + err.Error())
	}
	defer consumerGrp.Close() //nolint:errcheck
	go messaging.LogConsumerErrors("catalog", consumerGrp)

	// DLQ: message ที่ retry ครบแล้วยังพัง → product.events.dlq แทนการทิ้ง
	dlq, err := messaging.NewDeadLetterPublisher([]string{brokers})
	if err != nil {
		panic("failed to create kafka dead letter publisher: " + err.Error())
	}
	defer dlq.Close() //nolint:errcheck

	eventHandler := msghandler.NewProductEventHandler(cmdService)
	groupHandler := messaging.NewConsumerGroupHandler("catalog", eventHandler, dlq, messaging.DefaultRetryPolicy())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka consume loop — เข้ากลุ่มใหม่หลัง rebalance, backoff ตอน Kafka ล่ม, ออกเมื่อ shutdown
	go messaging.RunConsumerGroup(ctx, "catalog", consumerGrp, []string{kafkaTopic}, groupHandler)

	// --- HTTP server ---
	app := fiber.New()

	checker := health.New("catalog").Add("mongo", func(ctx context.Context) error {
		return mongoClient.Ping(ctx, nil)
	})
	checker.Register(app)

	handler := cataloghttp.NewCatalogHandler(queryService)

	// catalog routes — public, no auth required
	catalog := app.Group("/catalog")
	catalog.Get("/products", handler.SearchProducts)
	catalog.Get("/products/:productId", handler.GetProduct)
	catalog.Get("/variants/:variantId", handler.GetVariantInfo)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = defaultPort
	}

	logs.Info("Catalog Service starting on port " + port)
	err = checker.Serve(app, ":"+port)

	// HTTP ปิดแล้ว → หยุด Kafka consumer ต่อ แล้ว defer ปิด consumer group / DLQ / Mongo
	cancel()
	if err != nil {
		panic(err.Error()) // panic ยังรัน defer ก่อน exit ต่างจาก os.Exit
	}
}
