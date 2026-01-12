package config

import (
	"auth_service/internal/adapter/repository/postgres/entity"
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

func LoadConfig() *Config {
	_ = godotenv.Load()
	// _ = godotenv.Load("/internal/config/.env")

	getEnv := func(key, fallback string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return fallback
	}

	config := &Config{
		DBHost:     getEnv("DB_HOST_AUTH", "localhost"),
		DBPort:     getEnv("DB_PORT_AUTH", "5432"),
		DBUser:     getEnv("DB_USER_AUTH", "myuser"),
		DBPassword: getEnv("DB_PASSWORD_AUTH", "password"),
		DBName:     getEnv("DB_NAME_AUTH", "auth_db"),
	}

	return config
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)

}

func OpenDatabase(dsn string) *gorm.DB {
	db, err := database.ConnectPostgres(dsn)
	if err != nil {
		panic("failed to connect to database")
	}

	print(db)
	fmt.Println("Database connected!")

	// Auto-migrate the UserEntity schema
	err = db.AutoMigrate(&entity.UserEntity{}, &entity.SessionEntity{})
	if err != nil {
		panic("failed to migrate database")
	}

	fmt.Println("Database migration completed!")

	return db
}
