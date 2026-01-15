# Hexagonal Architecture Implementation

This project now follows **Hexagonal Architecture** (also known as Ports and Adapters) to achieve clean separation of concerns and better testability.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Primary Adapters                        │
│                  (Driving / Input Side)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │   HTTP   │  │  Worker  │  │   CLI    │                  │
│  │ Handlers │  │ Runtime  │  │  (future)│                  │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘                  │
│        │             │              │                        │
└────────┼─────────────┼──────────────┼────────────────────────┘
         │             │              │
         v             v              v
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│                  (Use Cases / Services)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │  Queue   │  │ Insights │  │  Worker  │                  │
│  │ Service  │  │ Service  │  │ Service  │                  │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘                  │
│        │             │              │                        │
└────────┼─────────────┼──────────────┼────────────────────────┘
         │             │              │
         v             v              v
┌─────────────────────────────────────────────────────────────┐
│                      Domain Layer                            │
│                  (Business Logic / Entities)                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │   Job    │  │ Insight  │  │  Worker  │                  │
│  │  Entity  │  │  Entity  │  │  Config  │                  │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘                  │
│        │             │              │                        │
│  ┌─────┴─────────────┴──────────────┴────┐                  │
│  │         Ports (Interfaces)             │                  │
│  │  - JobRepository                       │                  │
│  │  - QueueService                        │                  │
│  │  - AIService                           │                  │
│  └────────────────┬───────────────────────┘                  │
└───────────────────┼────────────────────────────────────────┘
                    │
                    v
┌─────────────────────────────────────────────────────────────┐
│                    Secondary Adapters                        │
│                  (Driven / Output Side)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Postgres │  │  Redis   │  │  Ollama  │  │ Metrics  │   │
│  │   Repo   │  │  Queue   │  │    AI    │  │ Service  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
internal/
├── domain/                      # Core business logic (no external dependencies)
│   ├── queue/
│   │   ├── job.go              # Job entity with business rules
│   │   └── ports.go            # Interfaces for repositories and services
│   ├── insights/
│   │   ├── insight.go          # Insight entity
│   │   └── ports.go            # Interfaces for AI service
│   └── worker/
│       ├── worker.go           # Worker domain logic
│       └── ports.go            # Job executor interface
│
├── application/                 # Use cases / orchestration
│   ├── queue/
│   │   └── service.go          # Queue use cases (CreateJob, GetJob, etc.)
│   ├── insights/
│   │   └── service.go          # Insights use cases (AnalyzeJob, etc.)
│   └── worker/
│       └── service.go          # Worker use cases (ProcessJob, etc.)
│
├── adapters/
│   ├── primary/                # Input adapters (driving)
│   │   └── http/
│   │       ├── queue_handlers.go
│   │       ├── insights_handlers.go
│   │       └── routes.go
│   │
│   └── secondary/              # Output adapters (driven)
│       ├── persistence/
│       │   ├── postgres_job_repository.go
│       │   ├── postgres_insight_repository.go
│       │   └── redis_queue_service.go
│       ├── ai/
│       │   └── ollama_service.go        # Local Ollama (phi3:mini model)
│       ├── insights/
│       │   └── http_client.go           # Remote insights API client
│       ├── metrics/
│       │   └── inmemory_metrics.go
│       └── executor/
│           └── default_executor.go
│
└── infrastructure/             # Cross-cutting concerns
    ├── config/
    │   └── config.go
    └── database/
        ├── postgres.go
        └── redis.go
```

## Key Principles

### 1. **Dependency Rule**
Dependencies point **inward**:
- **Adapters** depend on **Application**
- **Application** depends on **Domain**
- **Domain** has NO dependencies (pure business logic)

### 2. **Domain Layer (Core)**
- Contains pure business logic
- Entities have business rules and validation
- Defines ports (interfaces) that adapters must implement
- No external dependencies (no DB, HTTP, frameworks)

**Example:**
```go
// Domain entity with business rules
func (j *Job) CanRetry(maxAttempts int) bool {
    return j.Attempts < maxAttempts && j.Status == StatusFailed
}
```

### 3. **Application Layer (Use Cases)**
- Orchestrates domain objects
- Implements business workflows
- Uses ports defined in domain layer
- Independent of infrastructure details

**Example:**
```go
func (s *Service) CreateJob(ctx context.Context, cmd CreateJobCommand) (*queue.Job, error) {
    job, err := queue.NewJob(cmd.Queue, cmd.Type, payloadBytes) // Domain
    s.jobRepo.Create(ctx, job)                                   // Port
    s.queueService.Enqueue(ctx, job)                             // Port
    s.metrics.RecordJobCreated(job.Queue, job.Type)              // Port
    return job, nil
}
```

### 4. **Adapters Layer**

#### Primary Adapters (Input/Driving)
- HTTP handlers
- Worker runtime
- CLI commands (future)
- gRPC servers (future)

#### Secondary Adapters (Output/Driven)
- Database repositories (Postgres via Supabase)
- Queue service (Redis via Upstash with TLS)
- AI service (Ollama with phi3:mini model, 2.3GB, ~2-3 min analysis)
- HTTP insights client (for distributed deployment, 5-min timeout)
- Metrics service (in-memory)
- Job executors (default command executor)

### 5. **Infrastructure Layer**
- Configuration management
- Database connections
- Logging
- Cross-cutting concerns

## Benefits

### ✅ **Testability**
Each layer can be tested independently:
- **Domain**: Pure unit tests, no mocks needed
- **Application**: Mock the ports
- **Adapters**: Integration tests

### ✅ **Flexibility**
Easy to swap implementations:
- Switch from Postgres to MongoDB → just create new adapter
- Switch from Redis to RabbitMQ → just create new adapter
- Switch from Ollama to OpenAI → just create new adapter
- Switch from local AI to remote HTTP API → conditional initialization based on config

**Real example**: Worker service conditionally uses either:
- `OllamaAIService` adapter (local) when `insights_url` is empty
- `HTTPClient` adapter (remote) when `insights_url` is set

Both implement the same `AIService` port, so application layer doesn't change!

### ✅ **Maintainability**
- Clear boundaries between layers
- Business logic isolated from technical details
- Each component has a single responsibility

### ✅ **Scalability**
- Can extract adapters into microservices if needed
- Domain and application layers remain unchanged

## Service Separation Strategy

### Current: Monorepo with 3 Services ✅

All three services share the same codebase:
- `queue-core` - HTTP API for queue management
- `worker-runtime` - Background job processor
- `ai-insights-service` - AI analysis service

**Why keep them together:**
1. Shared domain models and business rules
2. Single deployment simplifies operations
3. Easier transaction management
4. Common infrastructure code
5. Lower overhead for a small team

### When to Separate into Microservices

Only separate if you have:
- **Different scaling needs** (e.g., insights service needs 10x resources)
- **Different teams** owning different services
- **Independent deployment cycles** required
- **Truly independent bounded contexts**

## Dependency Injection

All services use constructor-based dependency injection:

```go
// Domain ports are injected into application services
queueService := appQueue.NewService(
    jobRepo,        // Port implementation
    queueService,   // Port implementation
    metricsService, // Port implementation
)

// Application services are injected into adapters
queueHandlers := httpHandlers.NewQueueHandlers(queueService)
```

## Testing Strategy

### Domain Layer
```go
func TestJob_CanRetry(t *testing.T) {
    job := &Job{Status: StatusFailed, Attempts: 2}
    assert.True(t, job.CanRetry(3))
}
```

### Application Layer
```go
func TestService_CreateJob(t *testing.T) {
    mockRepo := &MockJobRepository{}
    service := NewService(mockRepo, ...)
    // Test use case
}
```

### Adapters Layer
```go
func TestPostgresJobRepository_Create(t *testing.T) {
    // Integration test with real database
}
```

## Migration from Old Structure

### Before (Layered Architecture)
```
internal/
├── queue/jobs/
│   ├── model.go       # Mixed domain + data access
│   └── repository.go  # Tightly coupled to Postgres
└── api/rest/
    └── handlers.go    # Contains business logic
```

### After (Hexagonal Architecture)
```
internal/
├── domain/queue/
│   ├── job.go         # Pure domain entity
│   └── ports.go       # Interfaces
├── application/queue/
│   └── service.go     # Use cases
└── adapters/
    ├── primary/http/
    │   └── queue_handlers.go  # HTTP adapter
    └── secondary/persistence/
        └── postgres_job_repository.go  # DB adapter
```

## Running the Services

### Build all services
```bash
go build -o bin/queue-core.exe ./cmd/queue-core
go build -o bin/worker-runtime.exe ./cmd/worker-runtime
go build -o bin/ai-insights-service.exe ./cmd/ai-insights-service
```

### Or use VS Code tasks
- Run task: `Build`
- Run task: `Run All`

## API Endpoints

### Queue API
- `POST /api/jobs` - Create a new job
- `GET /api/jobs?id={id}` - Get job by ID
- `GET /api/jobs?status={status}` - Get jobs by status
- `POST /api/jobs/retry?id={id}` - Retry a failed job
- `GET /api/dlq` - Get dead letter queue jobs
- `GET /api/metrics` - Get queue metrics

### Insights API
- `GET /api/insights?id={id}` - Get insight by ID
- `GET /api/insights?job_id={id}` - Get insight for a job
- `GET /api/insights` - List all insights
- `POST /api/insights/analyze?job_id={id}` - Analyze a job

## Future Enhancements

1. **Event-Driven Architecture**: Add domain events
2. **CQRS**: Separate read and write models
3. **API Gateway**: Add for service orchestration
4. **Service Discovery**: If scaling to microservices
5. **Message Bus**: For async communication between services
