// WHAT: Configuration loader สำหรับ order_service
// WHY อ่านจาก Environment Variable?
//   - 12-Factor App: config ใน env ไม่ใช่ hardcode ใน code
//   - Docker/K8s ฉีด env ได้ง่าย ไม่ต้อง rebuild image
//   - getEnv pattern: fallback ค่า default สำหรับ local development
package config

import (
	"database"
	"fmt"
	"order_service/internal/adapter/repository/postgres/entity"
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
	// Stripe — ได้จาก https://dashboard.stripe.com → Developers → API keys
	// Test:  STRIPE_SECRET_KEY=sk_test_xxxx, STRIPE_WEBHOOK_SECRET=whsec_xxxx
	// Live:  STRIPE_SECRET_KEY=sk_live_xxxx, STRIPE_WEBHOOK_SECRET=whsec_xxxx
	StripeSecretKey     string
	StripeWebhookSecret string
	CatalogServiceURL   string // http://catalog-service-app:3005
}

// Loadconfig โหลด config จาก .env file (local dev) หรือ environment variables (prod)
func Loadconfig() *Config {
	_ = godotenv.Load() // ไม่ panic ถ้าไม่มี .env file

	getEnv := func(key, fallback string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return fallback
	}

	return &Config{
		DBHost:              getEnv("DB_HOST_ORDER", "localhost"),
		DBPort:              getEnv("DB_PORT_ORDER", "5432"),
		DBUser:              getEnv("DB_USER_ORDER", "myuser"),
		DBPassword:          getEnv("DB_PASSWORD_ORDER", "mypassword"),
		DBName:              getEnv("DB_NAME_ORDER", "order_db"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		CatalogServiceURL:   getEnv("CATALOG_SERVICE_URL", "http://localhost:3005"),
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

// OpenDatabase เชื่อม DB และ run AutoMigrate สำหรับ order_service entities
// WHY AutoMigrate แทน migration file ใน local dev?
//   - AutoMigrate สะดวกสำหรับ local dev (ไม่ต้อง run migration script)
//   - Production ควรใช้ migration tool (golang-migrate) สำหรับ version control ของ schema
//   - migrations/ folder มี SQL files สำหรับ production deploy
func OpenDatabase(dsn string) *gorm.DB {
	db, err := database.ConnectPostgres(dsn)
	if err != nil {
		panic("failed to connect to order database: " + err.Error())
	}

	err = db.AutoMigrate(
		&entity.OrderEntity{},
		&entity.OrderItemEntity{},
		&entity.OutboxEventEntity{},
		&entity.InboxEventEntity{},
		&entity.PaymentEntity{},
	)
	if err != nil {
		panic("failed to migrate order database: " + err.Error())
	}

	return db
}
