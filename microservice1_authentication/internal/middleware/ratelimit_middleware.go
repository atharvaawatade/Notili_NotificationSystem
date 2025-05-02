package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterStore manages rate limiters per API key
var rateLimiterStore = struct {
	mu    sync.Mutex
	store map[string]*rate.Limiter
}{
	store: make(map[string]*rate.Limiter),
}

// getRateLimiter returns a rate limiter for the given API key, creating one if necessary
func getRateLimiter(apiKey string, r rate.Limit, b int) *rate.Limiter {
	rateLimiterStore.mu.Lock()
	defer rateLimiterStore.mu.Unlock()
	limiter, exists := rateLimiterStore.store[apiKey]
	if !exists {
		limiter = rate.NewLimiter(r, b)
		rateLimiterStore.store[apiKey] = limiter
	}
	return limiter
}

// RateLimitMiddleware returns a Gin middleware that rate limits requests per API key
func RateLimitMiddleware() gin.HandlerFunc {
	// Allow dynamic configuration via env vars (defaults: 10 req/sec, burst 20)
	ratePerSec, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_PER_SEC"))
	if ratePerSec <= 0 {
		ratePerSec = 10
	}
	burst, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_BURST"))
	if burst <= 0 {
		burst = 20
	}
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Optionally, you could use IP address or a global limiter here
			apiKey = c.ClientIP()
		}
		limiter := getRateLimiter(apiKey, rate.Limit(ratePerSec), burst)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
