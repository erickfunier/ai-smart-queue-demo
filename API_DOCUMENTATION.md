# AI Smart Queue - API Documentation

## Table of Contents
1. [System Architecture](#system-architecture)
2. [Component Block Diagram](#component-block-diagram)
3. [API Sequence Flows](#api-sequence-flows)
4. [Endpoint Reference](#endpoint-reference)

---

## System Architecture

### High-Level Architecture Diagram

```mermaid
graph TB
    subgraph "Client Layer"
        Client[Frontend Application]
        Postman[API Testing Tools]
    end

    subgraph "VM2: API & Worker<br/>163.176.239.253"
        QueueCore[Queue Core API<br/>:8080]
        Worker[Worker Runtime]
    end

    subgraph "VM1: AI Services<br/>163.176.243.66"
        InsightsAPI[AI Insights API<br/>:8082]
        Ollama[Ollama<br/>phi3:mini model]
    end

    subgraph "External Services"
        Postgres[(PostgreSQL<br/>Supabase)]
        Redis[(Redis<br/>Upstash)]
    end

    Client -->|HTTP| QueueCore
    Client -->|HTTP| InsightsAPI
    Postman -->|HTTP| QueueCore
    Postman -->|HTTP| InsightsAPI
    
    QueueCore -->|Store Jobs| Postgres
    QueueCore -->|Queue Jobs| Redis
    Worker -->|Dequeue| Redis
    Worker -->|HTTP Request| InsightsAPI
    Worker -->|Update Status| Postgres
    
    InsightsAPI -->|Store Insights| Postgres
    InsightsAPI -->|AI Analysis| Ollama
    
    style QueueCore fill:#4CAF50
    style Worker fill:#4CAF50
    style InsightsAPI fill:#2196F3
    style Ollama fill:#2196F3
    style Postgres fill:#FF9800
    style Redis fill:#F44336
```

---

## Component Block Diagram

### Hexagonal Architecture

```mermaid
graph TB
    subgraph "Queue Core Service (Port 8080)"
        subgraph "Inbound Adapters"
            HTTP1[HTTP Handlers]
        end
        
        subgraph "Application Layer"
            QueueService[Queue Service]
        end
        
        subgraph "Domain Layer"
            JobDomain[Job Domain]
        end
        
        subgraph "Outbound Adapters"
            JobRepo[Job Repository<br/>PostgreSQL]
            RedisQueue[Redis Queue Service]
            MetricsAdapter[Metrics Adapter]
        end
        
        HTTP1 --> QueueService
        QueueService --> JobDomain
        QueueService --> JobRepo
        QueueService --> RedisQueue
        QueueService --> MetricsAdapter
    end
    
    subgraph "Worker Runtime"
        subgraph "Application Layer Worker"
            WorkerService[Worker Service]
        end
        
        subgraph "Outbound Adapters Worker"
            Executor[Job Executor]
            HTTPInsights[HTTP Insights Client]
        end
        
        WorkerService --> Executor
        WorkerService --> HTTPInsights
    end
    
    subgraph "AI Insights Service (Port 8082)"
        subgraph "Inbound Adapters AI"
            HTTP2[HTTP Handlers]
        end
        
        subgraph "Application Layer AI"
            InsightsService[Insights Service]
        end
        
        subgraph "Domain Layer AI"
            InsightDomain[Insight Domain]
        end
        
        subgraph "Outbound Adapters AI"
            InsightRepo[Insight Repository<br/>PostgreSQL]
            OllamaAdapter[Ollama AI Service]
        end
        
        HTTP2 --> InsightsService
        InsightsService --> InsightDomain
        InsightsService --> InsightRepo
        InsightsService --> OllamaAdapter
    end
    
    HTTPInsights -.->|HTTP POST| HTTP2
    
    style JobDomain fill:#E8F5E9
    style InsightDomain fill:#E3F2FD
    style QueueService fill:#C8E6C9
    style InsightsService fill:#BBDEFB
```

---

## API Sequence Flows

### 1. Complete Job Lifecycle with AI Analysis

```mermaid
sequenceDiagram
    participant Client
    participant QueueAPI as Queue Core API<br/>:8080
    participant Redis
    participant Worker
    participant InsightsAPI as Insights API<br/>:8082
    participant Ollama
    participant Postgres

    Note over Client,Postgres: Job Creation Phase
    Client->>QueueAPI: POST /api/jobs<br/>{queue, type, payload}
    QueueAPI->>Postgres: Save job (status: pending)
    QueueAPI->>Redis: Enqueue job
    QueueAPI-->>Client: 201 Created<br/>{job_id, status: pending}

    Note over Client,Postgres: Job Processing Phase
    Worker->>Redis: Dequeue job
    Redis-->>Worker: Job data
    Worker->>Postgres: Update status to 'processing'
    Worker->>Worker: Execute job
    
    alt Job Fails
        Worker->>Worker: Job execution failed
        Worker->>InsightsAPI: POST /api/insights/analyze?job_id={id}
        
        Note over InsightsAPI,Ollama: AI Analysis Phase
        InsightsAPI->>Postgres: Check cached insight (GetByJobID)
        
        alt No cached insight
            InsightsAPI->>Postgres: Get job details
            InsightsAPI->>Ollama: Analyze failure<br/>(phi3:mini model)
            Ollama-->>InsightsAPI: AI diagnosis & recommendations
            InsightsAPI->>Postgres: Save insight
            InsightsAPI-->>Worker: 200 OK<br/>{diagnosis, recommendation}
        else Cached insight exists
            InsightsAPI->>Postgres: Retrieve cached insight
            InsightsAPI-->>Worker: 200 OK<br/>{cached insight}
        end
        
        Worker->>Postgres: Update job status to 'failed'
        Worker->>Redis: Re-enqueue if retries available
    else Job Succeeds
        Worker->>Postgres: Update status to 'completed'
    end

    Note over Client,Postgres: Client Retrieval Phase
    Client->>QueueAPI: GET /api/jobs/{id}
    QueueAPI->>Postgres: Get job with insight
    QueueAPI-->>Client: 200 OK<br/>{job + insight data}
```

### 2. Insights Query Flow

```mermaid
sequenceDiagram
    participant Client
    participant InsightsAPI as Insights API<br/>:8082
    participant Postgres

    Note over Client,Postgres: List All Insights
    Client->>InsightsAPI: GET /api/insights/
    InsightsAPI->>Postgres: SELECT * FROM insights<br/>ORDER BY created_at DESC
    Postgres-->>InsightsAPI: Insights array
    InsightsAPI-->>Client: 200 OK<br/>[{insights}]

    Note over Client,Postgres: Get Specific Insight
    Client->>InsightsAPI: GET /api/insights/{id}
    InsightsAPI->>Postgres: SELECT WHERE id = {id}
    Postgres-->>InsightsAPI: Insight record
    InsightsAPI-->>Client: 200 OK<br/>{insight}

    Note over Client,Postgres: Get Insight by Job ID
    Client->>InsightsAPI: GET /api/insights/?job_id={id}
    InsightsAPI->>Postgres: SELECT WHERE job_id = {id}
    Postgres-->>InsightsAPI: Insight record
    InsightsAPI-->>Client: 200 OK<br/>{insight}
```

### 3. Job Retry Flow

```mermaid
sequenceDiagram
    participant Client
    participant QueueAPI as Queue Core API<br/>:8080
    participant Postgres
    participant Redis

    Client->>QueueAPI: POST /api/jobs/retry<br/>{job_id}
    QueueAPI->>Postgres: Get job by ID
    
    alt Job exists and failed
        QueueAPI->>Postgres: Reset retry_count
        QueueAPI->>Postgres: Update status to 'pending'
        QueueAPI->>Redis: Re-enqueue job
        QueueAPI-->>Client: 200 OK<br/>{message: "Job re-queued"}
    else Job not found or not failed
        QueueAPI-->>Client: 400 Bad Request<br/>{error: "Cannot retry"}
    end
```

### 4. Dead Letter Queue Flow

```mermaid
sequenceDiagram
    participant Client
    participant QueueAPI as Queue Core API<br/>:8080
    participant Postgres

    Client->>QueueAPI: GET /api/dlq
    QueueAPI->>Postgres: SELECT WHERE retry_count >= max_retries
    Postgres-->>QueueAPI: Failed jobs array
    QueueAPI-->>Client: 200 OK<br/>[{failed jobs}]
```

---

## Endpoint Reference

### Base URLs

| Environment | Service | URL |
|------------|---------|-----|
| **Production** | Queue Core | `http://163.176.239.253:8080` |
| **Production** | AI Insights | `http://163.176.243.66:8082` |
| **Development** | Queue Core | `http://localhost:8080` |
| **Development** | AI Insights | `http://localhost:8082` |

### Queue Core API (Port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/jobs` | Create a new job |
| GET | `/api/jobs` | List jobs (with filters) |
| GET | `/api/jobs/{id}` | Get job by ID |
| POST | `/api/jobs/retry` | Retry a failed job |
| GET | `/api/dlq` | Get dead letter queue jobs |
| GET | `/api/metrics` | Get system metrics |
| GET | `/health` | Health check |

### AI Insights API (Port 8082)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/insights/` | List all insights |
| GET | `/api/insights/{id}` | Get insight by ID |
| GET | `/api/insights/?job_id={id}` | Get insight by job ID |
| POST | `/api/insights/analyze` | Trigger AI analysis for a job |
| GET | `/health` | Health check |

### Example Requests

#### Create Job
```bash
curl -X POST http://163.176.239.253:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "default",
    "type": "send-email",
    "payload": {"to": "user@example.com", "subject": "Hello"}
  }'
```

#### Get Job with Insights
```bash
curl http://163.176.239.253:8080/api/jobs/{job_id}
```

#### List Insights
```bash
curl http://163.176.243.66:8082/api/insights/
```

#### Trigger AI Analysis
```bash
curl -X POST "http://163.176.243.66:8082/api/insights/analyze?job_id={job_id}"
```

### Response Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (invalid input) |
| 404 | Not Found |
| 500 | Internal Server Error |

### Key Features

- **AI-Powered Analysis**: phi3:mini model (2.3GB, ~2-3 min analysis time)
- **Caching**: Prevents redundant AI analysis via GetByJobID check
- **Retry Logic**: Automatic retry with exponential backoff
- **Cross-VM Communication**: HTTP-based insights (5-min timeout)
- **Dead Letter Queue**: Failed jobs after max retries

### Performance Metrics

- **AI Analysis Time**: 48s - 2m50s (phi3:mini)
- **Model Size**: 2.3GB (96% smaller load than Mistral)
- **Timeout**: 5 minutes for AI analysis
- **Max Retries**: 3 attempts per job

---

## Converting to PDF

To convert this document to PDF:

### Option 1: VS Code Extension
1. Install "Markdown PDF" extension
2. Open this file
3. Press `Ctrl+Shift+P`
4. Type "Markdown PDF: Export (pdf)"

### Option 2: Pandoc
```bash
pandoc API_DOCUMENTATION.md -o API_DOCUMENTATION.pdf --pdf-engine=xelatex
```

### Option 3: GitHub
- View this file on GitHub
- Use browser "Print to PDF" feature

---

**Last Updated**: January 2, 2026
**API Version**: 1.0.0
