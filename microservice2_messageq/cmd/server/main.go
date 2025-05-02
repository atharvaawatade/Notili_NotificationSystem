package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/appointy/notli/microservice2_messageq/internal/handler"
	"github.com/appointy/notli/microservice2_messageq/internal/kafka"
	"github.com/appointy/notli/microservice2_messageq/internal/service"
	"github.com/appointy/notli/microservice2_messageq/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (optional)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// --- Dependency Initialization ---
	authClient := service.NewAuthClient()

	// Initialize Idempotency Store (PostgreSQL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Longer timeout for potential DB init
	defer cancel() // Ensure context cancellation

	pgStore, err := store.NewPostgresStore(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL store: %v", err)
	}
	defer func() {
		// Disconnect logic for PostgreSQL store
		ctxDisconnect, cancelDisconnect := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDisconnect()
		if err := pgStore.Disconnect(ctxDisconnect); err != nil {
			log.Printf("Error disconnecting PostgreSQL: %v", err)
		} else {
			log.Println("PostgreSQL disconnected.")
		}
	}()

	// Initialize Kafka Producer
	kafkaProducer, err := kafka.NewKafkaProducer()
	if err != nil {
		log.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		}
		log.Println("Kafka producer closed.")
	}()

	// --- Handler and Router Setup ---
	emailHandler := handler.NewEmailHandler(authClient, pgStore, kafkaProducer)

	app := gin.New()
	app.Use(gin.Logger())
	app.Use(gin.Recovery())

	// Health Check Endpoint
	app.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Register API routes using the injected handler
	handler.RegisterEmailRoutes(app, emailHandler)

	// --- Server Start and Graceful Shutdown ---
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "4000"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: app,
	}

	go func() {
		// Service connections
		log.Printf("Starting server on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the requests it is currently handling
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
