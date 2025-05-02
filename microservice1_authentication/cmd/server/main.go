package main

import (
	"log"
	"net/http"

	"github.com/appointy/notli/microservice1_authentication/internal/config"
	"github.com/appointy/notli/microservice1_authentication/internal/database"
	"github.com/appointy/notli/microservice1_authentication/internal/handler"
	"github.com/appointy/notli/microservice1_authentication/internal/middleware"
	"github.com/appointy/notli/microservice1_authentication/internal/models"
	"github.com/appointy/notli/microservice1_authentication/internal/repository"
	"github.com/appointy/notli/microservice1_authentication/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading environment variables from system")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Database Connection
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run Database Migrations
	log.Println("Running database migrations...")
	
	// Clean approach to avoid constraint issues
	db := database.DB
	
	// Drop tables if they exist
	if err := db.Exec("DROP TABLE IF EXISTS api_keys CASCADE").Error; err != nil {
		log.Fatalf("Failed to drop api_keys table: %v", err)
	}
	
	if err := db.Exec("DROP TABLE IF EXISTS users CASCADE").Error; err != nil {
		log.Fatalf("Failed to drop users table: %v", err)
	}
	
	// Now migrate the models
	if err := db.AutoMigrate(&models.User{}, &models.APIKey{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(database.DB)

	// Initialize session manager
	sessionManager, err := service.NewSessionManager(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg, sessionManager)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	validateHandler := handler.NewValidateHandler(apiKeyService)

	// Setup Gin
	router := gin.Default()

	// Add rate limiting middleware globally
	router.Use(middleware.RateLimitMiddleware())

	// Health check route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Serve static files
	router.Static("/static", "./web/static")
	
	// Redirect root to login page
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/login.html")
	})

	// Auth routes (public)
	authGroup := router.Group("/api/auth")
	authGroup.POST("/signup", authHandler.Register) // Handle user registration
	authGroup.POST("/login", authHandler.Login) // Handle user login
	authGroup.POST("/refresh", authHandler.Refresh) // Handle session refresh
	authGroup.POST("/logout", authHandler.Logout) // Handle user logout
	authGroup.POST("/validate", validateHandler.ValidateAPIKey) // Validate API keys
	
	// Proxy handler for cross-origin requests
	proxyHandler := handler.NewProxyHandler()
	router.POST("/api/proxy/email", proxyHandler.ProxyEmailRequest)

	// Protected routes
	apiGroup := router.Group("/api")
	apiGroup.Use(middleware.AuthMiddleware(authService))
	
	// API Key routes
	keyGroup := apiGroup.Group("/keys")
	keyGroup.POST("/", apiKeyHandler.CreateAPIKey)
	keyGroup.GET("/", apiKeyHandler.GetAPIKeys)
	keyGroup.DELETE("/:keyId", apiKeyHandler.DeleteAPIKey)

	// Start the server
	address := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("Server starting on %s", address)
	if err := router.Run(address); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
