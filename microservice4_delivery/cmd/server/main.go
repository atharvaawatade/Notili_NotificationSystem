package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/appointy/notli/microservice4_delivery/internal/config"
	"github.com/appointy/notli/microservice4_delivery/internal/handler"
	"github.com/appointy/notli/microservice4_delivery/internal/kafka"
	"github.com/appointy/notli/microservice4_delivery/internal/model"
	"github.com/appointy/notli/microservice4_delivery/internal/provider"
	"github.com/appointy/notli/microservice4_delivery/internal/provider/resend"
	"github.com/appointy/notli/microservice4_delivery/internal/repository"
	"github.com/appointy/notli/microservice4_delivery/internal/retry"
	"github.com/appointy/notli/microservice4_delivery/internal/service"
	"github.com/appointy/notli/microservice4_delivery/internal/template"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to PostgreSQL
	db, err := connectToDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create repositories
	emailRepo := repository.NewEmailRepository(db)

	// Initialize the provider registry
	providerRegistry := provider.NewProviderRegistry()
	providerRegistry.Register("resend", resend.ResendFactory)

	// Create the Resend provider with the verified domain email
	emailProvider, err := providerRegistry.Create("resend", map[string]string{
		"api_key":       cfg.ResendAPIKey,             // API key from config
		"domain":        cfg.ResendDomain,             // Domain for authenticated sends
		"verified_email": cfg.ResendVerifiedEmail,      // Verified email from the domain
		"detailed_logs": fmt.Sprintf("%t", cfg.ResendDetailedLogs), // Enable detailed logging
	})
	if err != nil {
		log.Fatalf("Failed to create Resend provider: %v", err)
	}

	log.Println("Using ONLY Resend as the email provider (Maileroo has been disabled)")
	defer emailProvider.Close()

	// Create the custom template engine with our green and white themed transactional templates
	templateOpts := &template.SetupOptions{
		MailjetAPIKey:    cfg.MailjetAPIKey,
		MailjetSecretKey: cfg.MailjetSecretKey,
	}
	
	// Use our new setup function that prioritizes in-house templates for transactional emails
	templateEngine, err := template.SetupTemplateEngine(templateOpts)
	if err != nil {
		log.Printf("Warning: Error setting up template engine: %v", err)
		// Fallback to basic engine if setup fails
		templateEngine = template.NewEngine(template.EngineOptions{
			TemplateDirs:   []string{"templates"},
			ReloadInterval: 5 * time.Minute,
		})
	}
	
	log.Println("Using in-house templates with green/white theme for transactional emails")
	
	// Log Mailjet configuration status (for future promotional emails)
	if cfg.EnableMailjetTemplates {
		log.Printf("Mailjet templates are configured for future promotional use. API Key: %s...", maskAPIKey(cfg.MailjetAPIKey))
	} else {
		log.Println("Mailjet templates are disabled. Using in-house templates for all emails.")
	}

	// Create the retry strategy
	retryStrategy := retry.DefaultStrategy(
		cfg.EmailMaxRetries,
		cfg.EmailRetryInitialDelay,
		cfg.EmailRetryMaxDelay,
	)

	// Create the email service
	emailService := service.NewEmailService(
		emailProvider,
		templateEngine,
		emailRepo,
		retryStrategy,
		10, // Number of worker goroutines
	)

	// Initialize the database
	if err := emailService.InitDB(ctx); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Start the email service
	if err := emailService.Start(ctx); err != nil {
		log.Fatalf("Failed to start email service: %v", err)
	}
	defer emailService.Stop()

	// Create Kafka consumer with improved configuration
	consumer := kafka.NewConsumer(
		cfg.KafkaBootstrapServers,
		cfg.KafkaConsumerGroup,
		cfg.KafkaInputTopic,
		cfg,
	)
	defer consumer.Close()

	// Start the consumer in a goroutine
	go func() {
		log.Printf("Starting Kafka consumer for topic: %s", cfg.KafkaInputTopic)
		if err := consumer.Consume(ctx, func(msgCtx context.Context, msg *model.KafkaMessage) error {
			return emailService.HandleMessage(msgCtx, msg)
		}); err != nil {
			log.Printf("Kafka consumer stopped with error: %v", err)
		}
	}()

	// Create HTTP server with Gin
	router := gin.Default()

	// Register handlers
	healthHandler := handler.NewHealthHandler()
	healthHandler.RegisterRoutes(router)

	emailHandler := handler.NewEmailHandler(emailService)
	emailHandler.RegisterRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerPort),
		Handler: router,
	}

	// Start the HTTP server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %d", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation (SIGINT or SIGTERM)
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Create a timeout context for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully shutdown the HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// connectToDatabase establishes a connection to PostgreSQL
// maskAPIKey returns a masked version of an API key for secure logging
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "[hidden]" 
	}
	return key[0:4] + "..." + key[len(key)-4:]
}

func connectToDatabase(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
