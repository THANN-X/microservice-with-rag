package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectPostgres establishes a connection to a PostgreSQL database using the provided DSN.
func ConnectPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DryRun: false,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
