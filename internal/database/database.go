package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init initializes the database connection
func Init(databaseURL string) error {
	if databaseURL == "" {
		return fmt.Errorf("database URL is required")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established successfully")

	// Auto-migrate the schema
	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return nil
}

// AutoMigrate runs database migrations
func AutoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&OTP{},
		&PartnerRequest{},
		&Partnership{},
		&Game{},
		&GameRequest{},
		&Play{},
	)
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
