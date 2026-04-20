package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rate-limited-api/internal/handlers"
	"rate-limited-api/internal/middleware"
	"rate-limited-api/internal/services"
	"rate-limited-api/pkg/config"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	gin.SetMode(cfg.Server.Mode)

	services, err := services.NewServices(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer services.Close()

	handlers := handlers.NewHandlers(services.RateLimiter)

	router := setupRouter(handlers)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	startServer(server, cfg)
}

func setupRouter(h *handlers.Handlers) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.SecurityHeaders())
	r.Use(gin.Recovery())

	r.POST("/request", h.HandleRequest)
	r.GET("/stats", h.HandleStats)

	return r
}

func startServer(server *http.Server, cfg *config.Config) {
	go func() {
		log.Printf("Starting server on port %s in %s mode", cfg.Server.Port, cfg.Server.Mode)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}
