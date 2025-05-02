package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	ServerPort int
	GRPCPort   int

	// Database settings
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// Kafka settings
	KafkaBootstrapServers string
	KafkaConsumerGroup    string
	KafkaInputTopic       string
	KafkaSessionTimeout   time.Duration
	KafkaHeartbeatInterval time.Duration
	KafkaReadTimeout      time.Duration
	KafkaWriteTimeout     time.Duration
	KafkaMaxWaitTime      time.Duration
	KafkaMinFetchBytes    int
	KafkaAutoOffsetReset  string

	// Maileroo settings (deprecated)
	MailerooAPIKey     string
	MailerooDomain     string
	MailerooFromEmail  string
	MailerooFromName   string

	// Resend settings
	ResendAPIKey       string
	ResendDomain      string
	ResendVerifiedEmail string // The verified email address for the domain
	ResendDetailedLogs bool

	// Mailjet template settings
	MailjetAPIKey    string
	MailjetSecretKey string
	EnableMailjetTemplates bool

	// Retry settings
	EmailMaxRetries       int
	EmailRetryInitialDelay time.Duration
	EmailRetryMaxDelay    time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file from multiple possible locations
	// First try the current directory
	err := godotenv.Load()
	
	// If that fails, try the project root
	if err != nil {
		_ = godotenv.Load("../../.env")
	}
	
	// If that fails too, try an absolute path (assuming standard project structure)
	_ = godotenv.Load("../../../.env")
	
	// Log the environment variables for debugging
	log.Printf("RESEND_API_KEY: %s", maskSecret(os.Getenv("RESEND_API_KEY")))
	log.Printf("RESEND_DOMAIN: %s", os.Getenv("RESEND_DOMAIN"))
	log.Printf("RESEND_VERIFIED_EMAIL: %s", os.Getenv("RESEND_VERIFIED_EMAIL"))

	config := &Config{}

	// Server settings
	serverPort, err := strconv.Atoi(getEnv("SERVER_PORT", "3002"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}
	config.ServerPort = serverPort

	grpcPort, err := strconv.Atoi(getEnv("GRPC_PORT", "3003"))
	if err != nil {
		return nil, fmt.Errorf("invalid GRPC_PORT: %w", err)
	}
	config.GRPCPort = grpcPort

	// Database settings
	config.DBHost = getEnv("DB_HOST", "localhost")
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	config.DBPort = dbPort
	config.DBName = getEnv("DB_NAME", "appointy")
	config.DBUser = getEnv("DB_USER", "postgres")
	config.DBPassword = getEnv("DB_PASSWORD", "")

	// Kafka settings
	config.KafkaBootstrapServers = getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	config.KafkaConsumerGroup = getEnv("KAFKA_CONSUMER_GROUP", "email-distribution-service")
	config.KafkaInputTopic = getEnv("KAFKA_INPUT_TOPIC", "distributed-notifications")
	
	// Parse Kafka timeouts
	sessionTimeout, err := strconv.Atoi(getEnv("KAFKA_SESSION_TIMEOUT", "60000"))
	if err != nil {
		return nil, fmt.Errorf("invalid KAFKA_SESSION_TIMEOUT: %w", err)
	}
	config.KafkaSessionTimeout = time.Duration(sessionTimeout) * time.Millisecond
	
	heartbeatInterval, err := strconv.Atoi(getEnv("KAFKA_HEARTBEAT_INTERVAL", "3000"))
	if err != nil {
		return nil, fmt.Errorf("invalid KAFKA_HEARTBEAT_INTERVAL: %w", err)
	}
	config.KafkaHeartbeatInterval = time.Duration(heartbeatInterval) * time.Millisecond
	
	readTimeout, err := strconv.Atoi(getEnv("KAFKA_READ_TIMEOUT", "30000"))
	if err != nil {
		return nil, fmt.Errorf("invalid KAFKA_READ_TIMEOUT: %w", err)
	}
	config.KafkaReadTimeout = time.Duration(readTimeout) * time.Millisecond
	
	writeTimeout, err := strconv.Atoi(getEnv("KAFKA_WRITE_TIMEOUT", "30000"))
	if err != nil {
		return nil, fmt.Errorf("invalid KAFKA_WRITE_TIMEOUT: %w", err)
	}
	config.KafkaWriteTimeout = time.Duration(writeTimeout) * time.Millisecond
	
	maxWaitTime, err := strconv.Atoi(getEnv("KAFKA_MAX_WAIT_TIME", "1000"))
	if err != nil {
		return nil, fmt.Errorf("invalid KAFKA_MAX_WAIT_TIME: %w", err)
	}
	config.KafkaMaxWaitTime = time.Duration(maxWaitTime) * time.Millisecond
	
	minFetchBytes, err := strconv.Atoi(getEnv("KAFKA_MIN_FETCH_BYTES", "1"))
	if err != nil {
		return nil, fmt.Errorf("invalid KAFKA_MIN_FETCH_BYTES: %w", err)
	}
	config.KafkaMinFetchBytes = minFetchBytes
	
	config.KafkaAutoOffsetReset = getEnv("KAFKA_AUTO_OFFSET_RESET", "earliest")

	// Maileroo settings (no longer required/enforced since we're using Resend)
	config.MailerooAPIKey = getEnv("MAILEROO_API_KEY", "")
	config.MailerooDomain = getEnv("MAILEROO_DOMAIN", "")
	config.MailerooFromEmail = getEnv("MAILEROO_FROM_EMAIL", "")
	config.MailerooFromName = getEnv("MAILEROO_FROM_NAME", "Notli")
	
	// Resend settings
	config.ResendAPIKey = getEnv("RESEND_API_KEY", "")
	if config.ResendAPIKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}
	
	config.ResendDomain = getEnv("RESEND_DOMAIN", "simplivu.com")
	config.ResendVerifiedEmail = getEnv("RESEND_VERIFIED_EMAIL", "simplivu@simplivu.com")
	config.ResendDetailedLogs = getEnv("RESEND_DETAILED_LOGS", "false") == "true"

	// Mailjet template settings
	config.MailjetAPIKey = getEnv("MAILJET_API_KEY", "cc35d2d982f49df106a504265ca30e7b")
	config.MailjetSecretKey = getEnv("MAILJET_SECRET_KEY", "771a11942eeac0bbfda1076dbac4c8ed")
	config.EnableMailjetTemplates = getEnv("ENABLE_MAILJET_TEMPLATES", "true") == "true"

	// Retry settings
	emailMaxRetries, err := strconv.Atoi(getEnv("EMAIL_MAX_RETRIES", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid EMAIL_MAX_RETRIES: %w", err)
	}
	config.EmailMaxRetries = emailMaxRetries

	emailRetryInitialDelay, err := strconv.Atoi(getEnv("EMAIL_RETRY_INITIAL_DELAY", "30"))
	if err != nil {
		return nil, fmt.Errorf("invalid EMAIL_RETRY_INITIAL_DELAY: %w", err)
	}
	config.EmailRetryInitialDelay = time.Duration(emailRetryInitialDelay) * time.Second

	emailRetryMaxDelay, err := strconv.Atoi(getEnv("EMAIL_RETRY_MAX_DELAY", "3600"))
	if err != nil {
		return nil, fmt.Errorf("invalid EMAIL_RETRY_MAX_DELAY: %w", err)
	}
	config.EmailRetryMaxDelay = time.Duration(emailRetryMaxDelay) * time.Second

	return config, nil
}

// getEnv reads an environment variable with a fallback default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// maskSecret returns a masked version of a secret string to avoid leaking sensitive information in logs
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "[masked]"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
