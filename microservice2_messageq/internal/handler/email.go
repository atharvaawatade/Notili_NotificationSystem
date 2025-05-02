package handler

import (
	"fmt"
	"net/http"
	"net/mail"
	"sync"
	"time"

	"github.com/appointy/notli/microservice2_messageq/internal/kafka"
	"github.com/appointy/notli/microservice2_messageq/internal/model"
	"github.com/appointy/notli/microservice2_messageq/internal/service"
	"github.com/appointy/notli/microservice2_messageq/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/patrickmn/go-cache"
)

// EmailRequest defines the expected structure for incoming email notifications.
type EmailRequest struct {
	IdempotencyKey string   `json:"idempotency_key" binding:"required,uuid4"`
	Recipient      string   `json:"recipient" binding:"required,email"`
	Subject        string   `json:"subject" binding:"required"`
	Body           string   `json:"body" binding:"required"`
	FromName       string   `json:"from_name,omitempty"`
	FromEmail      string   `json:"from_email,omitempty" binding:"omitempty,email"`
	ReplyTo        []string `json:"reply_to,omitempty"`
}

// EmailHandler holds dependencies for handling email requests with high-performance optimizations.
type EmailHandler struct {
	validate    *validator.Validate
	authClient  service.AuthClient
	store       store.IdempotencyStore
	producer    kafka.MessageProducer
	apiKeyCache *cache.Cache     // Cache for API key validation results
	requestPool sync.Pool        // Pool of pre-allocated request objects
	statsMutex  sync.RWMutex     // Mutex for protecting metrics
	requestStats map[string]int64 // Simple metrics for monitoring
}

// NewEmailHandler creates a new optimized EmailHandler with dependencies.
func NewEmailHandler(authClient service.AuthClient, store store.IdempotencyStore, producer kafka.MessageProducer) *EmailHandler {
	// Create API key cache with 30-minute expiration and 1-hour cleanup
	apiKeyCache := cache.New(30*time.Minute, 1*time.Hour)
	
	// Create handler with all dependencies
	handler := &EmailHandler{
		validate:    validator.New(),
		authClient:  authClient,
		store:       store,
		producer:    producer,
		apiKeyCache: apiKeyCache,
		requestStats: make(map[string]int64),
	}
	
	// Initialize request pool after creating the struct to avoid copying sync values
	handler.requestPool.New = func() interface{} {
		return &EmailRequest{}
	}
	
	return handler
}

// RegisterEmailRoutes sets up the routes for email notifications with performance monitoring.
func RegisterEmailRoutes(router *gin.Engine, h *EmailHandler) {
	// Create an API route group for better organization and middleware application
	apiGroup := router.Group("/v1/email")
	
	// Add basic rate limiting middleware to prevent extreme abuse
	apiGroup.Use(h.rateLimitMiddleware())
	
	// Route for handling email notifications with optimized handler
	apiGroup.POST("/notify", h.Notify)
	
	// Add health check and metrics endpoints
	apiGroup.GET("/health", h.healthCheck)
	apiGroup.GET("/metrics", h.metricsEndpoint)
	
	// Batch endpoint for higher throughput
	apiGroup.POST("/notify-batch", h.NotifyBatch)
}

// Notify handles incoming email notification requests with optimized performance.
func (h *EmailHandler) Notify(c *gin.Context) {
	// Track timing for metrics
	startTime := time.Now()
	defer func() {
		h.updateMetric("request_duration_ms", time.Since(startTime).Milliseconds())
		h.updateMetric("requests_processed", 1)
	}()

	// 1. API Key Validation with caching
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
		h.updateMetric("auth_failures", 1)
		return
	}

	// Check API key cache first to avoid external service calls
	if _, found := h.apiKeyCache.Get(apiKey); !found {
		// Key not in cache, validate with service
		if err := h.authClient.ValidateAPIKey(apiKey); err != nil {
			// Failed validation
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid or unauthorized API key"})
			h.updateMetric("auth_failures", 1)
			return
		}
		// Successful validation, cache the result
		h.apiKeyCache.Set(apiKey, true, cache.DefaultExpiration)
	}

	// 2. Input Binding and Validation with object pooling
	// Get request object from pool instead of allocating new one
	reqObj := h.requestPool.Get().(*EmailRequest)
	defer h.requestPool.Put(reqObj) // Return to pool when done
	
	// Reset fields to avoid data leakage between requests
	*reqObj = EmailRequest{}
	
	// Bind JSON to the pooled object
	if err := c.ShouldBindJSON(reqObj); err != nil {
		// Handle validation errors concisely
		var errResponse gin.H
		statuscode := http.StatusBadRequest
		
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			errors := make(map[string]string, len(validationErrs))
			for _, e := range validationErrs {
				errors[e.Field()] = fmt.Sprintf("Validation failed on '%s' tag", e.Tag())
			}
			errResponse = gin.H{"error": "Validation failed", "details": errors}
		} else {
			errResponse = gin.H{"error": "Invalid request body", "details": err.Error()}
		}
		
		c.JSON(statuscode, errResponse)
		h.updateMetric("validation_failures", 1)
		return
	}

	// Additional custom validation for ReplyTo emails
	if !validateEmailList(reqObj.ReplyTo) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": gin.H{"ReplyTo": "Invalid email format in reply_to list"}})
		h.updateMetric("validation_failures", 1)
		return
	}

	// 3. Idempotency Check (already optimized in store implementation)
	isDuplicate, err := h.store.Check(c.Request.Context(), reqObj.IdempotencyKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request due to internal storage error"})
		h.updateMetric("storage_errors", 1)
		return
	}
	if isDuplicate {
		c.JSON(http.StatusConflict, gin.H{"error": "Duplicate request", "idempotency_key": reqObj.IdempotencyKey})
		h.updateMetric("duplicate_requests", 1)
		return
	}

	// 4. Message Creation & Enqueueing - Optimize object creation
	// Pre-allocate map with exact capacity needed
	channelData := make(map[string]interface{}, 5)
	
	// Populate channel data map efficiently
	channelData["subject"] = reqObj.Subject
	channelData["body"] = reqObj.Body
	channelData["from_name"] = reqObj.FromName
	channelData["from_email"] = reqObj.FromEmail
	channelData["reply_to"] = reqObj.ReplyTo
	
	// Create notification message with the populated data
	msg := model.NotificationMessage{
		Channel:        "email",
		IdempotencyKey: reqObj.IdempotencyKey,
		Recipient:      reqObj.Recipient,
		MessageType:    "transactional",
		ChannelData:    channelData,
	}

	// Send to Kafka producer (already optimized for batching)
	if err := h.producer.Enqueue(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue notification for processing"})
		h.updateMetric("enqueue_errors", 1)
		return
	}

	// 5. Success Response
	c.JSON(http.StatusAccepted, gin.H{"message": "Notification accepted for processing", "idempotency_key": reqObj.IdempotencyKey})
	h.updateMetric("successful_requests", 1)
}

// validateEmailList checks if all emails in a slice are valid.
func validateEmailList(emails []string) bool {
	for _, email := range emails {
		if _, err := mail.ParseAddress(email); err != nil {
			return false
		}
	}
	return true
}

// NotifyBatch handles batch email notification requests for higher throughput.
func (h *EmailHandler) NotifyBatch(c *gin.Context) {
	startTime := time.Now()
	defer func() {
		h.updateMetric("batch_request_duration_ms", time.Since(startTime).Milliseconds())
		h.updateMetric("batch_requests_processed", 1)
	}()

	// 1. API Key Validation with caching
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
		h.updateMetric("auth_failures", 1)
		return
	}

	// Check API key cache first
	if _, found := h.apiKeyCache.Get(apiKey); !found {
		if err := h.authClient.ValidateAPIKey(apiKey); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid or unauthorized API key"})
			h.updateMetric("auth_failures", 1)
			return
		}
		h.apiKeyCache.Set(apiKey, true, cache.DefaultExpiration)
	}

	// 2. Parse batch request
	var requests []EmailRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch request", "details": err.Error()})
		h.updateMetric("validation_failures", 1)
		return
	}

	// Enforce batch size limits
	if len(requests) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Batch size exceeds maximum of 100 requests"})
		h.updateMetric("validation_failures", 1)
		return
	}

	// Process each request concurrently with worker pool
	results := make([]gin.H, len(requests))
	wg := sync.WaitGroup{}
	
	// Process batch with bounded parallelism
	semaphore := make(chan struct{}, 20) // Limit to 20 concurrent operations
	
	for i, req := range requests {
		wg.Add(1)
		go func(index int, request EmailRequest) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore
			
			// Validate request
			if !validateEmailList(request.ReplyTo) {
				results[index] = gin.H{"status": "error", "error": "Invalid reply_to emails", "idempotency_key": request.IdempotencyKey}
				return
			}
			
			// Check idempotency
			isDuplicate, err := h.store.Check(c.Request.Context(), request.IdempotencyKey)
			if err != nil {
				results[index] = gin.H{"status": "error", "error": "Storage error", "idempotency_key": request.IdempotencyKey}
				return
			}
			if isDuplicate {
				results[index] = gin.H{"status": "duplicate", "idempotency_key": request.IdempotencyKey}
				return
			}
			
			// Create and enqueue message
			channelData := make(map[string]interface{})
			channelData["subject"] = request.Subject
			channelData["body"] = request.Body
			channelData["from_name"] = request.FromName
			channelData["from_email"] = request.FromEmail
			channelData["reply_to"] = request.ReplyTo
			
			msg := model.NotificationMessage{
				Channel:        "email",
				IdempotencyKey: request.IdempotencyKey,
				Recipient:      request.Recipient,
				MessageType:    "transactional",
				ChannelData:    channelData,
			}
			
			if err := h.producer.Enqueue(msg); err != nil {
				results[index] = gin.H{"status": "error", "error": "Failed to enqueue message", "idempotency_key": request.IdempotencyKey}
				return
			}
			
			results[index] = gin.H{"status": "accepted", "idempotency_key": request.IdempotencyKey}
			h.updateMetric("successful_batch_items", 1)
		}(i, req)
	}
	
	wg.Wait()
	c.JSON(http.StatusAccepted, gin.H{"message": "Batch processed", "results": results})
	h.updateMetric("total_batch_items_processed", int64(len(requests)))
}

// rateLimitMiddleware provides a basic rate limiting layer
func (h *EmailHandler) rateLimitMiddleware() gin.HandlerFunc {
	// Simple token bucket rate limiter
	type client struct {
		tokens  float64
		lastSeen time.Time
	}
	
	const rate = 100.0 // tokens per second per IP
	const capacity = 200.0 // maximum burst
	
	clientsMu := &sync.RWMutex{}
	clients := make(map[string]*client)
	
	// Start cleanup goroutine
	go func() {
		for range time.Tick(time.Minute) {
			clientsMu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > time.Minute*5 {
					delete(clients, ip)
				}
			}
			clientsMu.Unlock()
		}
	}()
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		clientsMu.Lock()
		defer clientsMu.Unlock()
		
		cc, exists := clients[clientIP]
		if !exists {
			clients[clientIP] = &client{tokens: capacity, lastSeen: time.Now()}
			c.Next()
			return
		}
		
		// Calculate token refill
		now := time.Now()
		elapsed := now.Sub(cc.lastSeen).Seconds()
		cc.lastSeen = now
		
		// Refill tokens based on time passed
		cc.tokens += rate * elapsed
		if cc.tokens > capacity {
			cc.tokens = capacity
		}
		
		// If we have at least one token, proceed
		if cc.tokens >= 1 {
			cc.tokens--
			c.Next()
			return
		}
		
		// Rate limit exceeded
		h.updateMetric("rate_limit_exceeded", 1)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"retry_after": 1, // seconds until a token is available (minimum value)
		})
		c.Abort()
	}
}

// healthCheck provides a simple health check endpoint
func (h *EmailHandler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"version": "1.0",
		"timestamp": time.Now().Unix(),
	})
}

// metricsEndpoint provides basic metrics
func (h *EmailHandler) metricsEndpoint(c *gin.Context) {
	h.statsMutex.RLock()
	defer h.statsMutex.RUnlock()
	
	c.JSON(http.StatusOK, gin.H{
		"metrics": h.requestStats,
		"timestamp": time.Now().Unix(),
	})
}

// updateMetric safely updates request metrics
func (h *EmailHandler) updateMetric(name string, delta int64) {
	h.statsMutex.Lock()
	defer h.statsMutex.Unlock()
	
	h.requestStats[name] += delta
}
