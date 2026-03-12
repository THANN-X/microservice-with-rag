package config

import (
	"database"
	"fmt"
	"os"
	"product_service/internal/adapter/repository/postgres/entity"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func Loadconfig() *Config {
	_ = godotenv.Load()
	// _ = godotenv.Load("/internal/config/.env")

	getEnv := func(key, fallback string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return fallback
	}

	config := &Config{
		DBHost:     getEnv("DB_HOST_PRODUCT", "localhost"),
		DBPort:     getEnv("DB_PORT_PRODUCT", "5432"),
		DBUser:     getEnv("DB_USER_PRODUCT", "myuser"),
		DBPassword: getEnv("DB_PASSWORD_PRODUCT", "mypassword"),
		DBName:     getEnv("DB_NAME_PRODUCT", "product_db"),
	}

	return config
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

func OpenDatabase(dsn string) *gorm.DB {
	print(dsn)
	db, err := database.ConnectPostgres(dsn)
	if err != nil {
		panic("failed to connect to database")
	}

	print(db)
	fmt.Println("Database connected!")

	err = db.AutoMigrate(&entity.ProductEntity{}, &entity.ProductVariantEntity{}, &entity.AttributeEntity{}, &entity.AttributeValueEntity{}, &entity.InboxEventEntity{}, &entity.OutboxEventEntity{}, &entity.CategoryEntity{})

	if err != nil {
		panic("failed to migrate database")
	}

	fmt.Println("Database migration completed!")

	return db

}
