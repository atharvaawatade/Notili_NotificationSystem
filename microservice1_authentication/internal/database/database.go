package database

import (
	"fmt"
	"log"
	"time"

	"github.com/appointy/notli/microservice1_authentication/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect initializes the database connection
func Connect(cfg *config.Config) error {
	var err error
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	// Configure GORM logger
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)

	// Try connecting with retry logic
	maxRetries := 5
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: newLogger,
		})
		if err == nil {
			log.Println("Successfully connected to the database!")

			sqlDB, err := DB.DB()
			if err != nil {
				return fmt.Errorf("failed to get underlying sql.DB: %w", err)
			}

			// Set connection pool settings
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetMaxOpenConns(100)
			sqlDB.SetConnMaxLifetime(time.Hour)

			return nil // Connection successful
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}

// GetDB returns the initialized database connection
func GetDB() *gorm.DB {
	if DB == nil {
		log.Fatal("Database connection is not initialized. Call database.Connect() first.")
	}
	return DB
}
