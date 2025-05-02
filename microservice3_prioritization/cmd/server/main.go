package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/config"
	"github.com/appointy/notli/microservice3_prioritization/internal/handler"
	"github.com/appointy/notli/microservice3_prioritization/internal/kafka"
	"github.com/appointy/notli/microservice3_prioritization/internal/priority"
	"github.com/appointy/notli/microservice3_prioritization/internal/ratelimit"
	"github.com/appointy/notli/microservice3_prioritization/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis client for rate limiting
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	
	// Test Redis connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	pong, err := redisClient.Ping(pingCtx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v - rate limiting will be degraded", err)
	} else {
		log.Printf("Redis connection successful: %s - rate limiting enabled", pong)
	}

	// Initialize components
	consumer := kafka.NewConsumer(
		cfg.KafkaBootstrapServers,
		cfg.KafkaConsumerGroup,
		cfg.KafkaInputTopic,
	)

	producer := kafka.NewProducer(
		cfg.KafkaBootstrapServers,
		cfg.KafkaOutputTopic,
	)

	ruleEngine := priority.NewRuleEngine()
	rateLimiter := ratelimit.NewRateLimiter(redisClient)

	// Initialize priority service
	priorityService := service.NewPriorityService(
		consumer,
		producer,
		ruleEngine,
		rateLimiter,
		cfg.RateLimitTransactional,
		cfg.RateLimitPromotional,
	)

	// Start priority service
	if err := priorityService.Start(ctx); err != nil {
		log.Fatalf("Failed to start priority service: %v", err)
	}

	// Set up Gin router
	router := gin.Default()

	// Register health check handler
	healthHandler := handler.NewHealthHandler()
	handler.RegisterHealthRoutes(router, healthHandler)

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerPort),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		// Listen for signals
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down server...")
		
		// Create shutdown context
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		
		// Stop HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		
		// Stop priority service
		if err := priorityService.Stop(); err != nil {
			log.Printf("Priority service shutdown error: %v", err)
		}
		
		// Close Redis connection
		if err := redisClient.Close(); err != nil {
			log.Printf("Redis connection close error: %v", err)
		}
		
		// Cancel main context
		cancel()
	}()

	// Start server
	log.Printf("Starting server on port %d...", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
