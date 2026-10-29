package config

import (
	"database"
	"fmt"
	"os"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"

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

	// แปลง status ที่สะกดผิดแบบเก่า ("FAILD") ให้เป็น "FAILED" ตาม domain.OutboxStatusFailed
	// WHY ต้องแปลง? outbox query ข้าม aggregate ที่มี event FAILED เพื่อรักษาลำดับ
	//   ถ้า row เก่ายังเป็น "FAILD" query จะมองไม่เห็น แล้วส่ง event ถัดไปของ aggregate นั้นออกไปทั้งที่ไม่ควร
	// idempotent: รันซ้ำทุกครั้งที่ start ได้ ถ้าไม่มี row เก่าก็ไม่แก้อะไร
	// TODO: ย้ายไปเป็น migration file เมื่อเปลี่ยนจาก AutoMigrate ไปใช้ goose
	if err := db.Model(&entity.OutboxEventEntity{}).
		Where("status = ?", "FAILD").
		Update("status", domain.OutboxStatusFailed).Error; err != nil {
		panic("failed to normalize outbox status: " + err.Error())
	}

	return db

}
