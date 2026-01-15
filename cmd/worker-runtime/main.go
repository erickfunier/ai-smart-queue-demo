package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/ai"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/executor"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/insights"
	"github.com/erickfunier/ai-smart-queue/internal/adapters/outbound/persistence"
	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	appWorker "github.com/erickfunier/ai-smart-queue/internal/application/worker"
	domainInsights "github.com/erickfunier/ai-smart-queue/internal/domain/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/worker"
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

	// Initialize secondary adapters
	jobRepo := persistence.NewPostgresJobRepository(postgres.Pool)
	insightRepo := persistence.NewPostgresInsightRepository(postgres.Pool)
	queueService := persistence.NewRedisQueueService(redis.Client)
	jobExecutor := executor.NewDefaultJobExecutor(cfg)

	// Initialize insights service (use HTTP client if URL configured, otherwise local service)
	var aiSvc domainInsights.AIService
	if cfg.AI.InsightsURL != "" {
		// Use remote insights service via HTTP
		log.Printf("Using remote insights service: %s", cfg.AI.InsightsURL)
		aiSvc = insights.NewHTTPClient(cfg.AI.InsightsURL)
	} else {
		// Use local insights service with Ollama
		log.Println("Using local insights service with Ollama")
		aiSvc = ai.NewOllamaAIService(cfg.AI.OllamaURL)
	}

	insightsAppService := appInsights.NewService(insightRepo, jobRepo, aiSvc)

	// Create worker configuration
	workerConfig, err := worker.NewWorkerConfig(
		"default",
		cfg.Worker.MaxAttempts,
		cfg.Worker.BaseBackoffMs,
	)
	if err != nil {
		log.Fatalf("failed to create worker config: %v", err)
	}

	// Initialize worker application service
	workerService := appWorker.NewService(
		jobRepo,
		queueService,
		jobExecutor,
		insightsAppService,
		workerConfig,
	)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	log.Println("🚀 Worker Runtime service starting")
	log.Println("📦 Hexagonal Architecture initialized:")
	log.Println("   ├─ Domain: Business rules for job processing")
	log.Println("   ├─ Application: Worker orchestration")
	log.Println("   ├─ Adapters: Job executor, Queue service")
	log.Println("   └─ Infrastructure: Database, Config")

	// Start worker
	workerService.Start(ctx)
}
