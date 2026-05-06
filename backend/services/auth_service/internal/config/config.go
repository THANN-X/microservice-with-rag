package config

import (
	"auth_service/internal/adapter/repository/postgres/entity"
	"database"
	"fmt"
	"logs"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// What: Config เก็บค่าที่ต้องใช้เชื่อมต่อ Database และ external services
// Why:  รวม config ไว้ใน struct เดียวเพื่อส่งต่อได้ง่ายและ test ได้
type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	GoogleClientID string
}

// What: อ่าน environment variables แล้ว map ลง Config struct
// Why:  ใช้ env var แทนการ hardcode ทำให้ deploy ใน Docker/K8s ได้โดยไม่ต้องแก้โค้ด
// TODO: พิจารณาใช้ Viper หรือ envconfig library เพื่อ validate และ type-safe config
func LoadConfig() *Config {
	// What: โหลด .env file ถ้ามี (จะ skip ถ้าไม่มี ไม่ panic)
	_ = godotenv.Load()

	// What: helper function ที่อ่าน env หรือคืน fallback ถ้าไม่มี
	getEnv := func(key, fallback string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return fallback
	}

	// Why: แต่ละ service ใช้ prefix ต่างกัน (DB_HOST_AUTH) เพื่อให้อยู่ร่วมกันใน docker-compose ได้
	config := &Config{
		DBHost:         getEnv("DB_HOST_AUTH", "localhost"),
		DBPort:         getEnv("DB_PORT_AUTH", "5432"),
		DBUser:         getEnv("DB_USER_AUTH", "myuser"),
		DBPassword:     getEnv("DB_PASSWORD_AUTH", "mypassword"),
		DBName:         getEnv("DB_NAME_AUTH", "auth_db"),
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
	}

	return config
}

// What: สร้าง DSN string สำหรับเชื่อมต่อ Postgres
// Why:  แยก concern การสร้าง DSN ออกมา ทำให้ test และอ่านง่ายขึ้น
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

// What: เปิด connection ไปยัง Postgres และ auto-migrate schema ของ service นี้
// Why:  AutoMigrate ทำให้ไม่ต้องเขียน SQL DDL เอง และ schema sync กับ struct เสมอ
// TODO: ในระยะยาวควรเปลี่ยนเป็น migration tool (เช่น golang-migrate) เพื่อ rollback ได้
func OpenDatabase(dsn string) *gorm.DB {
	db, err := database.ConnectPostgres(dsn)
	if err != nil {
		// Why: ถ้า connect DB ไม่ได้ service ทำงานต่อไม่ได้เลย จึง panic แทน error handling
		panic("failed to connect to database")
	}

	logs.Info("Database connected!")

	// What: Auto-migrate สร้าง/อัปเดต ตาราง User, Session, Admin ให้ตรงกับ struct
	err = db.AutoMigrate(&entity.UserEntity{}, &entity.SessionEntity{}, &entity.AdminEntity{})
	if err != nil {
		panic("failed to migrate database")
	}

	logs.Info("Database migration completed!")

	return db
}
