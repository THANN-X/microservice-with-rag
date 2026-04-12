package config

import (
	"context"
	"logs"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	MongoURI string
	MongoDB  string
}

func LoadConfig() *Config {
	mongoURI := os.Getenv("MONGO_URI_CATALOG")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	mongoDB := os.Getenv("MONGO_DB_CATALOG")
	if mongoDB == "" {
		mongoDB = "catalog_db"
	}
	return &Config{
		MongoURI: mongoURI,
		MongoDB:  mongoDB,
	}
}

func ConnectMongo(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	logs.Info("Catalog Service: connected to MongoDB")
	return client, nil
}
