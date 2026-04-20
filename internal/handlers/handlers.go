package handlers

import (
	"fmt"
	"net/http"
	"time"

	"rate-limited-api/pkg/ratelimiter"

	"github.com/gin-gonic/gin"
)

type Request struct {
	UserID  string `json:"user_id" binding:"required"`
	Payload string `json:"payload" binding:"required"`
}

type Response struct {
	Message           string `json:"message"`
	UserID            string `json:"user_id,omitempty"`
	RemainingRequests int64  `json:"remaining_requests,omitempty"`
}

type ErrorResponse struct {
	Error      string `json:"error"`
	Limit      int64  `json:"limit,omitempty"`
	Current    int64  `json:"current,omitempty"`
	ResetTime  int64  `json:"reset_time,omitempty"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

type Handlers struct {
	rateLimiter *ratelimiter.RateLimiter
}

func NewHandlers(rateLimiter *ratelimiter.RateLimiter) *Handlers {
	return &Handlers{
		rateLimiter: rateLimiter,
	}
}

func (h *Handlers) HandleRequest(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	allowed, currentCount, resetTime, err := h.rateLimiter.CheckRateLimitWithLocalCache(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	if !allowed {
		retryAfter := int(time.Until(time.Unix(resetTime, 0)).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}

		c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
		c.JSON(http.StatusTooManyRequests, ErrorResponse{
			Error:      "Rate limit exceeded",
			Limit:      5,
			Current:    currentCount,
			ResetTime:  resetTime,
			RetryAfter: retryAfter,
		})
		return
	}

	if err := h.rateLimiter.IncrementTotalRequests(req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Message:           "Request processed successfully",
		UserID:            req.UserID,
		RemainingRequests: 5 - currentCount,
	})
}

func (h *Handlers) HandleStats(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "user_id parameter is required"})
		return
	}

	stats, err := h.rateLimiter.GetUserStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
