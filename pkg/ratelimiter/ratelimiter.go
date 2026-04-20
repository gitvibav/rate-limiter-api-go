package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	client   *redis.Client
	ctx      context.Context
	mu       sync.RWMutex
	requests map[string]*UserRequestTracker
}

type UserRequestTracker struct {
	mu           sync.Mutex
	currentCount int64
	lastReset    time.Time
}

type Stats struct {
	UserID      string `json:"user_id"`
	TotalReqs   int64  `json:"total_requests"`
	CurrentReqs int64  `json:"current_requests"`
	ResetTime   int64  `json:"reset_time"`
}

type Config struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

func NewRateLimiter(cfg Config) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	})

	ctx := context.Background()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return &RateLimiter{
		client:   rdb,
		ctx:      ctx,
		requests: make(map[string]*UserRequestTracker),
	}
}

func (rl *RateLimiter) CheckRateLimit(userID string) (bool, int64, int64, error) {
	key := fmt.Sprintf("rate_limit:%s", userID)

	pipe := rl.client.Pipeline()
	incrCmd := pipe.Incr(rl.ctx, key)
	pipe.Expire(rl.ctx, key, time.Minute)

	_, err := pipe.Exec(rl.ctx)
	if err != nil {
		return false, 0, 0, fmt.Errorf("redis pipeline error: %w", err)
	}

	currentCount := incrCmd.Val()
	resetTime := time.Now().Add(time.Minute).Unix()

	if currentCount > 5 {
		return false, currentCount, resetTime, nil
	}

	return true, currentCount, resetTime, nil
}

func (rl *RateLimiter) CheckRateLimitWithLocalCache(userID string) (bool, int64, int64, error) {
	rl.mu.RLock()
	tracker, exists := rl.requests[userID]
	rl.mu.RUnlock()

	if exists {
		tracker.mu.Lock()
		now := time.Now()

		if now.Sub(tracker.lastReset) >= time.Minute {
			tracker.currentCount = 0
			tracker.lastReset = now
		}

		tracker.currentCount++
		currentCount := tracker.currentCount

		tracker.mu.Unlock()

		if currentCount > 5 {
			return false, currentCount, now.Add(time.Minute).Unix(), nil
		}

		return true, currentCount, now.Add(time.Minute).Unix(), nil
	}

	return rl.CheckRateLimit(userID)
}

func (rl *RateLimiter) GetUserStats(userID string) (*Stats, error) {
	key := fmt.Sprintf("rate_limit:%s", userID)
	currentCount, err := rl.client.Get(rl.ctx, key).Int64()
	if err == redis.Nil {
		currentCount = 0
	} else if err != nil {
		return nil, fmt.Errorf("failed to get current count: %w", err)
	}

	totalKey := fmt.Sprintf("total_requests:%s", userID)
	totalCount, err := rl.client.Get(rl.ctx, totalKey).Int64()
	if err == redis.Nil {
		totalCount = 0
	} else if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	ttl, err := rl.client.TTL(rl.ctx, key).Result()
	if err != nil {
		ttl = time.Minute
	}

	resetTime := time.Now().Add(ttl).Unix()

	return &Stats{
		UserID:      userID,
		TotalReqs:   totalCount,
		CurrentReqs: currentCount,
		ResetTime:   resetTime,
	}, nil
}

func (rl *RateLimiter) IncrementTotalRequests(userID string) error {
	totalKey := fmt.Sprintf("total_requests:%s", userID)
	if err := rl.client.Incr(rl.ctx, totalKey).Err(); err != nil {
		return fmt.Errorf("failed to increment total requests: %w", err)
	}
	return nil
}

func (rl *RateLimiter) ResetUserRateLimit(userID string) error {
	key := fmt.Sprintf("rate_limit:%s", userID)
	if err := rl.client.Del(rl.ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	rl.mu.Lock()
	if tracker, exists := rl.requests[userID]; exists {
		tracker.mu.Lock()
		tracker.currentCount = 0
		tracker.lastReset = time.Now()
		tracker.mu.Unlock()
	}
	rl.mu.Unlock()

	return nil
}

func (rl *RateLimiter) Close() error {
	return rl.client.Close()
}
