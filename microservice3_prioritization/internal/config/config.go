package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// Server
	ServerPort int

	// Kafka
	KafkaBootstrapServers string
	KafkaConsumerGroup    string
	KafkaInputTopic       string
	KafkaOutputTopic      string

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// Rate Limiting
	RateLimitTransactional int
	RateLimitPromotional   int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPortStr := getEnv("DB_PORT", "5432")
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	dbName := getEnv("DB_NAME", "appointy")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")

	// Server configuration
	serverPortStr := getEnv("SERVER_PORT", "3001")
	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}

	// Kafka configuration
	kafkaBootstrapServers := getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	kafkaConsumerGroup := getEnv("KAFKA_CONSUMER_GROUP", "prioritization-service")
	kafkaInputTopic := getEnv("KAFKA_INPUT_TOPIC", "prioritized-notifications")
	kafkaOutputTopic := getEnv("KAFKA_OUTPUT_TOPIC", "distributed-notifications")

	// Redis configuration
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPortStr := getEnv("REDIS_PORT", "6379")
	redisPort, err := strconv.Atoi(redisPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDBStr := getEnv("REDIS_DB", "0")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	// Rate limiting configuration
	rateLimitTransactionalStr := getEnv("RATE_LIMIT_TRANSACTIONAL", "1000")
	rateLimitTransactional, err := strconv.Atoi(rateLimitTransactionalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_TRANSACTIONAL: %w", err)
	}
	rateLimitPromotionalStr := getEnv("RATE_LIMIT_PROMOTIONAL", "100")
	rateLimitPromotional, err := strconv.Atoi(rateLimitPromotionalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_PROMOTIONAL: %w", err)
	}

	return &Config{
		// Database
		DBHost:     dbHost,
		DBPort:     dbPort,
		DBName:     dbName,
		DBUser:     dbUser,
		DBPassword: dbPassword,

		// Server
		ServerPort: serverPort,

		// Kafka
		KafkaBootstrapServers: kafkaBootstrapServers,
		KafkaConsumerGroup:    kafkaConsumerGroup,
		KafkaInputTopic:       kafkaInputTopic,
		KafkaOutputTopic:      kafkaOutputTopic,

		// Redis
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,

		// Rate Limiting
		RateLimitTransactional: rateLimitTransactional,
		RateLimitPromotional:   rateLimitPromotional,
	}, nil
}

// getEnv gets environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
