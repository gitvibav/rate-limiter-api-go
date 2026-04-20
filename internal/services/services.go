package services

import (
	"rate-limited-api/pkg/config"
	"rate-limited-api/pkg/ratelimiter"
)

type Services struct {
	RateLimiter *ratelimiter.RateLimiter
}

func NewServices(cfg *config.Config) (*Services, error) {
	rateLimiter := ratelimiter.NewRateLimiter(ratelimiter.Config{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
	})

	return &Services{
		RateLimiter: rateLimiter,
	}, nil
}

func (s *Services) Close() error {
	if s.RateLimiter != nil {
		return s.RateLimiter.Close()
	}
	return nil
}
