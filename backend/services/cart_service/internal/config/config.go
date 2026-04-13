package config

import (
	"cart_service/internal/adapter/repository/postgres/entity"
	"database"
	"fmt"
	"os"

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
	getEnv := func(key, fallback string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return fallback
	}
	return &Config{
		DBHost:     getEnv("DB_HOST_CART", "localhost"),
		DBPort:     getEnv("DB_PORT_CART", "5432"),
		DBUser:     getEnv("DB_USER_CART", "myuser"),
		DBPassword: getEnv("DB_PASSWORD_CART", "mypassword"),
		DBName:     getEnv("DB_NAME_CART", "cart_db"),
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func OpenDatabase(dsn string) *gorm.DB {
	db, err := database.ConnectPostgres(dsn)
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	if err := db.AutoMigrate(
		&entity.CartEntity{},
		&entity.CartItemEntity{},
	); err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	return db
}
