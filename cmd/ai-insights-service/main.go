package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	httpHandlers "github.com/erickfunier/ai-smart-queue/internal/adapters/inbound/http"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/ai"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/persistence"
	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
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

	// Initialize secondary adapters
	insightRepo := persistence.NewPostgresInsightRepository(postgres.Pool)
	jobRepo := persistence.NewPostgresJobRepository(postgres.Pool)
	aiService := ai.NewOllamaAIService(cfg.AI.OllamaURL)

	// Initialize application service
	insightsAppService := appInsights.NewService(insightRepo, jobRepo, aiService)

	// Initialize HTTP handlers
	insightsHandlers := httpHandlers.NewInsightsHandlers(insightsAppService)

	// Setup routes
	mux := http.NewServeMux()
	httpHandlers.RegisterInsightsRoutes(mux, insightsHandlers)

	// Add health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	addr := fmt.Sprintf(":%d", 8082) // AI Insights runs on 8082
	log.Printf("🚀 AI Insights service running on %s", addr)
	log.Println("📦 Hexagonal Architecture initialized:")
	log.Println("   ├─ Domain: Insight business logic")
	log.Println("   ├─ Application: Analysis orchestration")
	log.Println("   ├─ Adapters: HTTP handlers, AI service")
	log.Println("   └─ Infrastructure: Database, Config")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
