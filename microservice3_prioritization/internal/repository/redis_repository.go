package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisRepository handles interaction with Redis
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// GetUserMetadata retrieves user-specific metadata like tier and preferences
func (r *RedisRepository) GetUserMetadata(ctx context.Context, userID string) (map[string]string, error) {
	key := fmt.Sprintf("user:metadata:%s", userID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user metadata: %w", err)
	}

	// If result is empty, return default metadata
	if len(result) == 0 {
		return map[string]string{
			"tier": "standard", // Default tier
		}, nil
	}

	return result, nil
}

// StoreProcessingStats records processing statistics for analytics
func (r *RedisRepository) StoreProcessingStats(ctx context.Context, messageType, channel string, processingTime time.Duration, priorityScore int) error {
	// Get current timestamp
	now := time.Now().Unix()
	
	// Create stats entry
	stats := map[string]interface{}{
		"message_type":    messageType,
		"channel":         channel,
		"processing_time": processingTime.Milliseconds(),
		"priority_score":  priorityScore,
		"timestamp":       now,
	}
	
	// Add to processing stats list with expiry (30 days)
	key := fmt.Sprintf("stats:processing:%d", now/60) // Group by minute
	pipe := r.client.Pipeline()
	pipe.HMSet(ctx, key, stats)
	pipe.Expire(ctx, key, 30*24*time.Hour)
	
	// Also increment counters for quick stats lookup
	countKey := fmt.Sprintf("stats:count:%s:%s:%d", messageType, channel, now/3600) // Group by hour
	pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, countKey, 30*24*time.Hour)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store processing stats: %w", err)
	}
	
	return nil
}

// GetSystemLoad determines current system load for adaptive rate limiting
func (r *RedisRepository) GetSystemLoad(ctx context.Context) (float64, error) {
	// In a real implementation, this would retrieve system metrics from Redis
	// such as queue depths, processing times, etc.
	
	// For now, return a default value
	return 1.0, nil
}

// Close closes the Redis repository
func (r *RedisRepository) Close() error {
	return r.client.Close()
}
