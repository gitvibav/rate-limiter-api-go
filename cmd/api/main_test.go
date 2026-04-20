package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"rate-limited-api/internal/handlers"
	"rate-limited-api/internal/services"
	"rate-limited-api/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("QUEUE_WORKERS", "2")

	code := m.Run()
	os.Exit(code)
}

func setupTestRouter() (*gin.Engine, *services.Services) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Addr:         "localhost:6379",
			Password:     "",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
		},
		Queue: config.QueueConfig{
			Workers:    2,
			QueueSize:  100,
			MaxRetries: 3,
		},
	}

	services, err := services.NewServices(cfg)
	if err != nil {
		panic(err)
	}

	handlers := handlers.NewHandlers(services.RateLimiter)

	router := gin.New()
	router.POST("/request", handlers.HandleRequest)
	router.GET("/stats", handlers.HandleStats)

	return router, services
}

func TestRequestEndpoint_Success(t *testing.T) {
	router, services := setupTestRouter()
	defer services.Close()

	payload := map[string]string{
		"user_id": "test_user_123",
		"payload": "test payload",
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/request", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Request processed successfully", response["message"])
	assert.Equal(t, "test_user_123", response["user_id"])
}

func TestRequestEndpoint_RateLimit(t *testing.T) {
	router, services := setupTestRouter()
	defer services.Close()

	payload := map[string]string{
		"user_id": "rate_limited_user",
		"payload": "test payload",
	}

	jsonData, _ := json.Marshal(payload)

	for i := 0; i < 6; i++ {
		req, _ := http.NewRequest("POST", "/request", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if i < 5 {
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request %d should be rate limited", i+1)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "Rate limit exceeded", response["error"])
			assert.Equal(t, float64(5), response["limit"])
		}
	}
}

func TestRequestEndpoint_InvalidJSON(t *testing.T) {
	router, services := setupTestRouter()
	defer services.Close()

	req, _ := http.NewRequest("POST", "/request", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStatsEndpoint(t *testing.T) {
	router, services := setupTestRouter()
	defer services.Close()

	payload := map[string]string{
		"user_id": "stats_user",
		"payload": "test payload",
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/request", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req, _ = http.NewRequest("GET", "/stats?user_id=stats_user", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "stats_user", response["user_id"])
	assert.Greater(t, response["total_requests"], float64(0))
}
