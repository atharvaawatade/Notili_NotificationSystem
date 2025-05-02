package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/appointy/notli/microservice3_prioritization/internal/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client, func()) {
	// Start a mini redis server for testing
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create a redis client that connects to the mini redis server
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Return the mini redis server, the client, and a cleanup function
	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return mr, client, cleanup
}

func TestRateLimiter_AllowMessage(t *testing.T) {
	_, client, cleanup := setupTestRedis(t)
	defer cleanup()

	limiter := NewRateLimiter(client)
	ctx := context.Background()

	// Test transactional message - should always be allowed
	info := &model.RateLimitInfo{
		UserID:      "user1",
		MessageType: "transactional",
		Channel:     "email",
		Timestamp:   time.Now(),
	}

	allowed, _, err := limiter.AllowMessage(ctx, info, 100)
	assert.NoError(t, err)
	assert.True(t, allowed, "Transactional message should be allowed")

	// Make multiple rapid requests for transactional messages - all should pass
	for i := 0; i < 10; i++ {
		allowed, _, err := limiter.AllowMessage(ctx, info, 100)
		assert.NoError(t, err)
		assert.True(t, allowed, "Transactional messages should always be allowed with high limits")
	}

	// Test promotional message - should be allowed initially
	promoInfo := &model.RateLimitInfo{
		UserID:      "user1",
		MessageType: "promotional",
		Channel:     "email",
		Timestamp:   time.Now(),
	}

	allowed, _, err = limiter.AllowMessage(ctx, promoInfo, 10)
	assert.NoError(t, err)
	assert.True(t, allowed, "First promotional message should be allowed")

	// Make multiple rapid requests for promotional messages - should hit rate limit
	var hitRateLimit bool
	for i := 0; i < 50; i++ {
		allowed, _, err := limiter.AllowMessage(ctx, promoInfo, 10)
		assert.NoError(t, err)
		if !allowed {
			hitRateLimit = true
			break
		}
	}

	assert.True(t, hitRateLimit, "Should eventually hit rate limit for promotional messages")

	// Test different users - shouldn't affect each other's rate limits
	otherUserInfo := &model.RateLimitInfo{
		UserID:      "user2",
		MessageType: "promotional",
		Channel:     "email",
		Timestamp:   time.Now(),
	}

	allowed, _, err = limiter.AllowMessage(ctx, otherUserInfo, 10)
	assert.NoError(t, err)
	assert.True(t, allowed, "Different user should have separate rate limit")

	// Test different channels - should have different rate limits
	smsInfo := &model.RateLimitInfo{
		UserID:      "user1",
		MessageType: "promotional",
		Channel:     "sms",
		Timestamp:   time.Now(),
	}

	allowed, _, err = limiter.AllowMessage(ctx, smsInfo, 5)
	assert.NoError(t, err)
	assert.True(t, allowed, "Different channel should have separate rate limit")
}

func TestGetTimeOfDayMultiplier(t *testing.T) {
	_, client, cleanup := setupTestRedis(t)
	defer cleanup()

	limiter := NewRateLimiter(client)
	
	// Test that multiplier is always within expected range
	multiplier := limiter.getTimeOfDayMultiplier()
	assert.GreaterOrEqual(t, multiplier, 0.5, "Multiplier should be at least 0.5")
	assert.LessOrEqual(t, multiplier, 1.0, "Multiplier should be at most 1.0")
}

func TestGetChannelMultiplier(t *testing.T) {
	_, client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	limiter := NewRateLimiter(client)
	
	tests := map[string]float64{
		"email":    1.0,
		"sms":      0.2,
		"push":     0.5,
		"unknown":  0.3,
	}
	
	for channel, expected := range tests {
		assert.Equal(t, expected, limiter.getChannelMultiplier(channel),
			"Channel %s should have multiplier %f", channel, expected)
	}
}
