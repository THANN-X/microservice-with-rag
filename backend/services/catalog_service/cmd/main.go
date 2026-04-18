package main

import (
	cataloghttp "catalog_service/internal/adapter/handler/http"
	"catalog_service/internal/adapter/messaging/consumer"
	msghandler "catalog_service/internal/adapter/messaging/handler"
	mongorepo "catalog_service/internal/adapter/repository/mongo"
	"catalog_service/internal/config"
	"catalog_service/internal/core/service/command"
	"catalog_service/internal/core/service/query"
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"logs"
)

const (
	kafkaTopic     = "product.events"
	consumerGroup  = "catalog-service-group"
	defaultPort    = "3005"
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

	eventHandler := msghandler.NewProductEventHandler(cmdService)
	groupHandler := consumer.NewConsumerGroupHandler(eventHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka consume loop — restart on rebalance
	go func() {
		for {
			if err := consumerGrp.Consume(ctx, []string{kafkaTopic}, groupHandler); err != nil {
				logs.Error("catalog: kafka consume error: " + err.Error())
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// --- HTTP server ---
	app := fiber.New()

	handler := cataloghttp.NewCatalogHandler(queryService)

	// catalog routes — public, no auth required
	catalog := app.Group("/catalog")
	catalog.Get("/products", handler.SearchProducts)
	catalog.Get("/products/:productId", handler.GetProduct)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = defaultPort
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logs.Info("Catalog Service starting on port " + port)
		if err := app.Listen(":" + port); err != nil {
			logs.Error(err)
		}
	}()

	<-quit
	logs.Info("Catalog Service shutting down...")
	cancel()
	if err := app.Shutdown(); err != nil {
		logs.Error(err)
	}
}
