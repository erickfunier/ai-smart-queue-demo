package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	httpHandlers "github.com/erickfunier/ai-smart-queue/internal/adapters/inbound/http"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/ai"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/metrics"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/persistence"
	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	appQueue "github.com/erickfunier/ai-smart-queue/internal/application/queue"
	"github.com/erickfunier/ai-smart-queue/internal/infrastructure/config"
	"github.com/erickfunier/ai-smart-queue/internal/infrastructure/database"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize infrastructure - database connections
	postgres, err := database.NewPostgresConnection(cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("postgres connection error: %v", err)
	}
	defer postgres.Close()

	if err := postgres.Ping(context.Background()); err != nil {
		log.Fatalf("postgres ping error: %v", err)
	}
	log.Println("✅ Connected to Postgres")

	redis := database.NewRedisConnection(cfg.Redis.Addr, cfg.Redis.URL, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.TLSSkipVerify)
	defer redis.Close()

	if err := redis.Ping(context.Background()); err != nil {
		log.Fatalf("redis ping error: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// Initialize secondary adapters (output ports implementations)
	jobRepo := persistence.NewPostgresJobRepository(postgres.Pool)
	insightRepo := persistence.NewPostgresInsightRepository(postgres.Pool)
	queueService := persistence.NewRedisQueueService(redis.Client)
	metricsService := metrics.NewInMemoryMetricsService()
	aiService := ai.NewOllamaAIService(cfg.AI.OllamaURL)

	// Initialize application services (use cases)
	queueAppService := appQueue.NewService(jobRepo, queueService, metricsService)
	insightsAppService := appInsights.NewService(insightRepo, jobRepo, aiService)

	// Initialize primary adapters (input ports / HTTP handlers)
	queueHandlers := httpHandlers.NewQueueHandlers(queueAppService, insightsAppService)
	insightsHandlers := httpHandlers.NewInsightsHandlers(insightsAppService)

	// Setup HTTP routes
	mux := http.NewServeMux()
	httpHandlers.RegisterQueueRoutes(mux, queueHandlers)
	httpHandlers.RegisterInsightsRoutes(mux, insightsHandlers)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("🚀 Queue Core service running on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
