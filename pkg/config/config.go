package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Redis    RedisConfig
	Queue    QueueConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
	Mode         string
}

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

type QueueConfig struct {
	Workers    int
	QueueSize  int
	MaxRetries int
}

type LoggingConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getEnvInt("READ_TIMEOUT", 10),
			WriteTimeout: getEnvInt("WRITE_TIMEOUT", 10),
			IdleTimeout:  getEnvInt("IDLE_TIMEOUT", 60),
			Mode:         getEnv("GIN_MODE", "release"),
		},
		Redis: RedisConfig{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 100),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 10),
			MaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
		},
		Queue: QueueConfig{
			Workers:    getEnvInt("QUEUE_WORKERS", 10),
			QueueSize:  getEnvInt("QUEUE_SIZE", 1000),
			MaxRetries: getEnvInt("QUEUE_MAX_RETRIES", 3),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis address cannot be empty")
	}
	if c.Queue.Workers <= 0 {
		return fmt.Errorf("queue workers must be positive")
	}
	if c.Queue.QueueSize <= 0 {
		return fmt.Errorf("queue size must be positive")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
