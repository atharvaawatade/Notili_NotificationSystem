package ratelimit

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/go-redis/redis/v8"
)

// RateLimiter handles adaptive rate limiting for notifications
type RateLimiter struct {
	redisClient *redis.Client
}

// NewRateLimiter creates a new rate limiter with Redis
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
	}
}

// AllowMessage determines if a message should be allowed based on rate limiting rules
func (r *RateLimiter) AllowMessage(ctx context.Context, info *model.RateLimitInfo, baseRate int) (bool, time.Duration, error) {
	// If Redis client is nil, always allow messages
	if r.redisClient == nil {
		// Bypass rate limiting when Redis is not available
		log.Println("Rate limiting bypassed - Redis client is nil")
		return true, 0, nil
	}

	// For promotional messages, apply stricter rate limiting
	if info.MessageType == "promotional" {
		return r.applyAdaptiveRateLimit(ctx, info, baseRate)
	}

	// For transactional messages, allow with minimal limiting
	// We still track usage but with a much higher limit
	return r.applyTransactionalRateLimit(ctx, info, baseRate*10)
}

// applyAdaptiveRateLimit applies rate limiting for promotional messages
func (r *RateLimiter) applyAdaptiveRateLimit(ctx context.Context, info *model.RateLimitInfo, baseRate int) (bool, time.Duration, error) {
	key := fmt.Sprintf("rate_limit:%s:%s:%s", info.MessageType, info.Channel, info.UserID)
	
	// Adaptive limits based on time of day and system load
	finalRate, burst := r.getAdaptiveRateLimits(info.Channel, baseRate)
	
	// Apply token bucket algorithm with Redis
	return r.tokenBucketAllow(ctx, key, finalRate, burst)
}

// applyTransactionalRateLimit applies minimal rate limiting for transactional messages
func (r *RateLimiter) applyTransactionalRateLimit(ctx context.Context, info *model.RateLimitInfo, baseRate int) (bool, time.Duration, error) {
	key := fmt.Sprintf("rate_limit:transactional:%s:%s", info.Channel, info.UserID)
	
	// High limits for transactional messages
	return r.tokenBucketAllow(ctx, key, baseRate, baseRate/2)
}

// getAdaptiveRateLimits calculates adaptive rate limits based on time and load
func (r *RateLimiter) getAdaptiveRateLimits(channel string, baseRate int) (int, int) {
	// Apply time-of-day adjustment (reduce rate during peak hours)
	timeMultiplier := r.getTimeOfDayMultiplier()
	
	// Apply system load adjustment (reduce rate during high load)
	loadMultiplier := r.getSystemLoadMultiplier()
	
	// Apply channel-specific adjustments
	channelMultiplier := r.getChannelMultiplier(channel)
	
	// Calculate final rate
	finalRate := int(float64(baseRate) * timeMultiplier * loadMultiplier * channelMultiplier)
	
	// Calculate burst capacity (allow bursts of 1.5x the rate)
	burst := int(float64(finalRate) * 1.5)
	
	return finalRate, burst
}

// getTimeOfDayMultiplier returns a multiplier based on time of day
func (r *RateLimiter) getTimeOfDayMultiplier() float64 {
	hour := time.Now().Hour()
	
	// Reduce rates during peak hours (9 AM - 6 PM)
	if hour >= 9 && hour <= 18 {
		return 0.8
	}
	
	// Reduce rates even more during super-peak (12 PM - 2 PM)
	if hour >= 12 && hour <= 14 {
		return 0.6
	}
	
	// Allow higher rates during off-peak hours
	return 1.0
}

// getSystemLoadMultiplier returns a multiplier based on system load
// In a real implementation, this would check system metrics
func (r *RateLimiter) getSystemLoadMultiplier() float64 {
	// For now, return a constant, but in a real implementation
	// this would check metrics like CPU usage, memory, queue sizes, etc.
	return 1.0
}

// getChannelMultiplier returns a multiplier specific to each channel
func (r *RateLimiter) getChannelMultiplier(channel string) float64 {
	switch channel {
	case "email":
		return 1.0 // Baseline
	case "sms":
		return 0.2 // More restrictive for SMS
	case "push":
		return 0.5 // Medium restriction for push
	default:
		return 0.3 // Conservative default
	}
}

// tokenBucketAllow implements the token bucket algorithm with Redis
func (r *RateLimiter) tokenBucketAllow(ctx context.Context, key string, rate, burst int) (bool, time.Duration, error) {
	now := time.Now().UnixNano()
	
	// Redis script for token bucket algorithm
	script := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = 1
		
		local lastUpdate = tonumber(redis.call("get", key..":last") or 0)
		local tokens = tonumber(redis.call("get", key..":tokens") or burst)
		
		-- Calculate time since last update in seconds
		local elapsed = math.max(0, now - lastUpdate) / 1000000000
		
		-- Refill tokens based on elapsed time, but don't exceed burst size
		tokens = math.min(burst, tokens + (elapsed * rate))
		
		-- Calculate wait time if not enough tokens
		local allowed = tokens >= requested
		local waitTime = 0
		
		if allowed then
			-- Consume token
			tokens = tokens - requested
			redis.call("set", key..":tokens", tokens)
			redis.call("set", key..":last", now)
			redis.call("expire", key..":tokens", 600)  -- Set expiration for cleanup
			redis.call("expire", key..":last", 600)     -- Set expiration for cleanup
			return {1, 0}  -- Allowed, no wait
		else
			-- Calculate wait time in nanoseconds
			waitTime = ((requested - tokens) / rate) * 1000000000
			return {0, waitTime}  -- Not allowed, return wait time
		end
	`)
	
	// Execute the Redis script
	result, err := script.Run(ctx, r.redisClient, []string{key}, rate, burst, now).Result()
	if err != nil {
		return false, 0, err
	}
	
	// Parse the result
	results, ok := result.([]interface{})
	if !ok || len(results) < 2 {
		return false, 0, fmt.Errorf("invalid result from rate limit script")
	}
	
	// Check if allowed
	allowed, ok := results[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("invalid allowed result type")
	}
	
	// Get wait time
	waitTime, ok := results[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf("invalid wait time result type")
	}
	
	return allowed == 1, time.Duration(waitTime), nil
}
