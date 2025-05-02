package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Database settings
	DBHost         string
	DBPort         string
	DBName         string
	DBUser         string
	DBPassword     string
	
	// Server settings
	ServerHost     string
	ServerPort     string
	
	// JWT settings
	JWTSecret      string
	JWTExpiryHours int
	
	// Redis settings
	RedisHost     string
	RedisPort     string
	RedisPassword string
	
	// Session settings
	SessionExpiryHours      int
	InactivityTimeoutMinutes int
	RefreshTokenExpiryHours int
	SecureCookies           bool
	CookieDomainName        string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Ensure .env is loaded (especially useful if run from different directories/tests)
	_ = godotenv.Load()

	// Parse JWT expiry hours
	jwtExpiryHoursStr := getEnv("JWT_EXPIRY_HOURS", "24")
	jwtExpiryHours, err := strconv.Atoi(jwtExpiryHoursStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}

	// Parse session expiry settings
	sessionExpiryHoursStr := getEnv("SESSION_EXPIRY_HOURS", "72") // 3 days by default
	sessionExpiryHours, err := strconv.Atoi(sessionExpiryHoursStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SESSION_EXPIRY_HOURS: %w", err)
	}

	inactivityTimeoutStr := getEnv("INACTIVITY_TIMEOUT_MINUTES", "30") // 30 minutes by default
	inactivityTimeout, err := strconv.Atoi(inactivityTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid INACTIVITY_TIMEOUT_MINUTES: %w", err)
	}

	refreshTokenExpiryStr := getEnv("REFRESH_TOKEN_EXPIRY_HOURS", "168") // 7 days by default
	refreshTokenExpiry, err := strconv.Atoi(refreshTokenExpiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REFRESH_TOKEN_EXPIRY_HOURS: %w", err)
	}

	// Parse secure cookies setting
	secureCookiesStr := getEnv("SECURE_COOKIES", "false")
	secureCookies, err := strconv.ParseBool(secureCookiesStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SECURE_COOKIES: %w", err)
	}

	cfg := &Config{
		// Database settings
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBName:         getEnv("DB_NAME", "appointy"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", ""),

		// Server settings
		ServerHost:     getEnv("SERVER_HOST", "localhost"),
		ServerPort:     getEnv("SERVER_PORT", "3000"),

		// JWT settings
		JWTSecret:      getEnv("JWT_SECRET", ""),
		JWTExpiryHours: jwtExpiryHours,

		// Redis settings
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		// Session settings
		SessionExpiryHours:      sessionExpiryHours,
		InactivityTimeoutMinutes: inactivityTimeout,
		RefreshTokenExpiryHours: refreshTokenExpiry,
		SecureCookies:           secureCookies,
		CookieDomainName:        getEnv("COOKIE_DOMAIN", ""),
	}

	// Validation
	if cfg.DBPassword == "" {
		fmt.Println("Warning: DB_PASSWORD environment variable not set.")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable must be set")
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
