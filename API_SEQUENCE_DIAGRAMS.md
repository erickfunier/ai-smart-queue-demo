# API Sequence Diagrams

This document contains sequence diagrams for all API endpoints in the AI Smart Queue system.

## Table of Contents
1. [Queue Endpoints](#queue-endpoints)
   - [POST /api/jobs - Create Job](#post-apijobs---create-job)
   - [GET /api/jobs - List Jobs](#get-apijobs---list-jobs)
   - [GET /api/jobs/{id} - Get Job by ID](#get-apijobsid---get-job-by-id)
   - [POST /api/jobs/retry - Retry Job](#post-apijobsretry---retry-job)
   - [GET /api/dlq - Get Dead Letter Queue Jobs](#get-apidlq---get-dead-letter-queue-jobs)
   - [GET /api/metrics - Get Metrics](#get-apimetrics---get-metrics)
2. [Insights Endpoints](#insights-endpoints)
   - [GET /api/insights - List Insights](#get-apiinsights---list-insights)
   - [GET /api/insights/{id} - Get Insight by ID](#get-apiinsightsid---get-insight-by-id)
   - [GET /api/insights?job_id={id} - Get Insight by Job ID](#get-apiinsightsjob_idid---get-insight-by-job-id)
   - [POST /api/insights/analyze - Analyze Job](#post-apiinsightsanalyze---analyze-job)

---

## Queue Endpoints

### POST /api/jobs - Create Job

Creates a new job in the queue system.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Queue Service
    participant Job Repository
    participant Redis Queue

    Client->>HTTP Handler: POST /api/jobs<br/>{queue, type, payload}
    HTTP Handler->>HTTP Handler: Validate request body
    HTTP Handler->>Queue Service: CreateJob(CreateJobCommand)
    Queue Service->>Queue Service: Create Job domain object
    Queue Service->>Job Repository: Save(job)
    Job Repository->>Job Repository: Store in PostgreSQL
    Job Repository-->>Queue Service: Job saved
    Queue Service->>Redis Queue: Enqueue(job)
    Redis Queue-->>Queue Service: Job enqueued
    Queue Service-->>HTTP Handler: Job created
    HTTP Handler->>HTTP Handler: Build JobResponse
    HTTP Handler-->>Client: 201 Created<br/>JobResponse
```

**Request:**
```json
{
  "queue": "email-queue",
  "type": "send-email",
  "payload": {...}
}
```

**Response:**
```json
{
  "id": "uuid",
  "queue": "email-queue",
  "type": "send-email",
  "status": "pending",
  "attempts": 0,
  "payload": {...},
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### GET /api/jobs - List Jobs

Lists jobs with optional filtering by status and queue.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Queue Service
    participant Job Repository

    Client->>HTTP Handler: GET /api/jobs?status=pending&limit=50
    HTTP Handler->>HTTP Handler: Parse query parameters<br/>(status, queue, limit, offset)
    
    alt Status filter provided
        HTTP Handler->>Queue Service: GetJobsByStatus(status, limit)
        Queue Service->>Job Repository: FindByStatus(status, limit)
        Job Repository-->>Queue Service: []Job
        Queue Service-->>HTTP Handler: []Job
    else No status filter
        HTTP Handler->>HTTP Handler: Return empty array<br/>(GetAllJobs not implemented)
    end
    
    HTTP Handler->>HTTP Handler: Build []JobResponse
    HTTP Handler-->>Client: 200 OK<br/>[]JobResponse
```

**Response:**
```json
[
  {
    "id": "uuid",
    "queue": "email-queue",
    "type": "send-email",
    "status": "pending",
    "attempts": 0,
    "payload": {...},
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

---

### GET /api/jobs/{id} - Get Job by ID

Retrieves a specific job by its ID, including insights if the job has failed.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Queue Service
    participant Job Repository
    participant Insights Service
    participant Insight Repository

    Client->>HTTP Handler: GET /api/jobs/{id}
    HTTP Handler->>HTTP Handler: Extract & validate ID
    HTTP Handler->>Queue Service: GetJob(id)
    Queue Service->>Job Repository: FindByID(id)
    Job Repository-->>Queue Service: Job
    Queue Service-->>HTTP Handler: Job
    
    alt Job status is FAILED
        HTTP Handler->>Insights Service: GetInsightByJobID(id)
        Insights Service->>Insight Repository: FindByJobID(id)
        Insight Repository-->>Insights Service: Insight
        Insights Service-->>HTTP Handler: Insight
        HTTP Handler->>HTTP Handler: Include insight in response
    end
    
    HTTP Handler->>HTTP Handler: Build JobResponse
    HTTP Handler-->>Client: 200 OK<br/>JobResponse with optional Insight
```

**Response (with insight):**
```json
{
  "id": "uuid",
  "queue": "email-queue",
  "type": "send-email",
  "status": "failed",
  "attempts": 3,
  "payload": {...},
  "error": "Connection timeout",
  "insight": {
    "id": "insight-uuid",
    "job_id": "uuid",
    "diagnosis": "...",
    "recommendation": "...",
    "suggested_fix": {...}
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### POST /api/jobs/retry - Retry Job

Retries a failed or dead-lettered job.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Queue Service
    participant Job Repository
    participant Redis Queue

    Client->>HTTP Handler: POST /api/jobs/retry?id={id}
    HTTP Handler->>HTTP Handler: Extract & validate ID
    HTTP Handler->>Queue Service: RetryJob(id, maxAttempts)
    Queue Service->>Job Repository: FindByID(id)
    Job Repository-->>Queue Service: Job
    
    alt Job can be retried
        Queue Service->>Queue Service: Reset job status<br/>& attempts
        Queue Service->>Job Repository: Update(job)
        Job Repository-->>Queue Service: Job updated
        Queue Service->>Redis Queue: Enqueue(job)
        Redis Queue-->>Queue Service: Job enqueued
        Queue Service-->>HTTP Handler: Success
        HTTP Handler-->>Client: 200 OK<br/>{status: "retrying"}
    else Job cannot be retried
        Queue Service-->>HTTP Handler: Error
        HTTP Handler-->>Client: 500 Error
    end
```

**Response:**
```json
{
  "status": "retrying"
}
```

---

### GET /api/dlq - Get Dead Letter Queue Jobs

Retrieves jobs from the dead letter queue with pagination.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Queue Service
    participant Job Repository

    Client->>HTTP Handler: GET /api/dlq?limit=50&offset=0
    HTTP Handler->>HTTP Handler: Parse pagination params<br/>(limit, offset)
    HTTP Handler->>Queue Service: GetDLQJobs(limit, offset)
    Queue Service->>Job Repository: FindDLQJobs(limit, offset)
    Job Repository->>Job Repository: Query jobs with status=dead_letter
    Job Repository-->>Queue Service: ([]Job, total count)
    Queue Service-->>HTTP Handler: ([]Job, total)
    HTTP Handler->>HTTP Handler: Build response with pagination
    HTTP Handler-->>Client: 200 OK<br/>{jobs, total, limit, offset}
```

**Response:**
```json
{
  "jobs": [
    {
      "id": "uuid",
      "queue": "email-queue",
      "status": "dead_letter",
      "attempts": 3,
      "error": "Max retries exceeded",
      ...
    }
  ],
  "total": 42,
  "limit": 50,
  "offset": 0
}
```

---

### GET /api/metrics - Get Metrics

Retrieves queue metrics and statistics.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Queue Service
    participant Metrics Repository

    Client->>HTTP Handler: GET /api/metrics
    HTTP Handler->>Queue Service: GetMetrics()
    Queue Service->>Metrics Repository: GetMetrics()
    Metrics Repository->>Metrics Repository: Retrieve in-memory metrics
    Metrics Repository-->>Queue Service: Metrics data
    Queue Service-->>HTTP Handler: Metrics
    HTTP Handler-->>Client: 200 OK<br/>Metrics JSON
```

**Response:**
```json
{
  "total_jobs": 1000,
  "pending_jobs": 50,
  "processing_jobs": 10,
  "completed_jobs": 920,
  "failed_jobs": 15,
  "dead_letter_jobs": 5,
  "average_processing_time": 1250,
  "success_rate": 0.92
}
```

---

## Insights Endpoints

### GET /api/insights - List Insights

Lists all insights with pagination.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Insights Service
    participant Insight Repository

    Client->>HTTP Handler: GET /api/insights?limit=50&offset=0
    HTTP Handler->>HTTP Handler: Parse pagination params<br/>(limit, offset)
    HTTP Handler->>Insights Service: ListInsights(limit, offset)
    Insights Service->>Insight Repository: FindAll(limit, offset)
    Insight Repository-->>Insights Service: []Insight
    Insights Service-->>HTTP Handler: []Insight
    HTTP Handler->>HTTP Handler: Build []InsightResponse
    HTTP Handler-->>Client: 200 OK<br/>[]InsightResponse
```

**Response:**
```json
[
  {
    "id": "uuid",
    "job_id": "job-uuid",
    "diagnosis": "Connection timeout after 30 seconds",
    "recommendation": "Increase timeout to 60 seconds",
    "suggested_fix": {
      "timeout_seconds": 60,
      "max_retries": 5,
      "payload_patch": {...}
    },
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

---

### GET /api/insights/{id} - Get Insight by ID

Retrieves a specific insight by its ID.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Insights Service
    participant Insight Repository

    Client->>HTTP Handler: GET /api/insights/{id}
    HTTP Handler->>HTTP Handler: Extract & validate ID
    HTTP Handler->>Insights Service: GetInsight(id)
    Insights Service->>Insight Repository: FindByID(id)
    Insight Repository-->>Insights Service: Insight
    Insights Service-->>HTTP Handler: Insight
    HTTP Handler->>HTTP Handler: Build InsightResponse
    HTTP Handler-->>Client: 200 OK<br/>InsightResponse
```

**Response:**
```json
{
  "id": "uuid",
  "job_id": "job-uuid",
  "diagnosis": "Connection timeout after 30 seconds",
  "recommendation": "Increase timeout to 60 seconds",
  "suggested_fix": {
    "timeout_seconds": 60,
    "max_retries": 5,
    "payload_patch": {...}
  },
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### GET /api/insights?job_id={id} - Get Insight by Job ID

Retrieves the insight associated with a specific job.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Insights Service
    participant Insight Repository

    Client->>HTTP Handler: GET /api/insights?job_id={job_id}
    HTTP Handler->>HTTP Handler: Extract & validate job_id
    HTTP Handler->>Insights Service: GetInsightByJobID(job_id)
    Insights Service->>Insight Repository: FindByJobID(job_id)
    Insight Repository-->>Insights Service: Insight
    Insights Service-->>HTTP Handler: Insight
    HTTP Handler->>HTTP Handler: Build InsightResponse
    HTTP Handler-->>Client: 200 OK<br/>InsightResponse
```

**Response:** Same as Get Insight by ID

---

### POST /api/insights/analyze - Analyze Job

Triggers AI analysis of a failed job to generate insights.

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Handler
    participant Insights Service
    participant Job Repository
    participant AI Service (Ollama)
    participant Insight Repository

    Client->>HTTP Handler: POST /api/insights/analyze?job_id={id}
    HTTP Handler->>HTTP Handler: Extract & validate job_id
    HTTP Handler->>Insights Service: AnalyzeJobFailure(job_id)
    Insights Service->>Job Repository: FindByID(job_id)
    Job Repository-->>Insights Service: Job
    
    Insights Service->>Insights Service: Build analysis context<br/>(error, attempts, payload)
    Insights Service->>AI Service (Ollama): AnalyzeFailure(context)
    AI Service (Ollama)->>AI Service (Ollama): Call LLM API<br/>(generate diagnosis & recommendations)
    AI Service (Ollama)-->>Insights Service: Analysis result
    
    Insights Service->>Insights Service: Create Insight domain object
    Insights Service->>Insight Repository: Save(insight)
    Insight Repository-->>Insights Service: Insight saved
    Insights Service-->>HTTP Handler: Insight
    
    HTTP Handler->>HTTP Handler: Build InsightResponse
    HTTP Handler-->>Client: 201 Created<br/>InsightResponse
```

**Response:**
```json
{
  "id": "uuid",
  "job_id": "job-uuid",
  "diagnosis": "Connection timeout after 30 seconds. The external API at https://api.example.com is experiencing high latency.",
  "recommendation": "Increase the timeout configuration to 60 seconds and implement exponential backoff for retries.",
  "suggested_fix": {
    "timeout_seconds": 60,
    "max_retries": 5,
    "payload_patch": {
      "retry_strategy": "exponential_backoff",
      "initial_delay_ms": 1000
    }
  },
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

## Architecture Notes

All endpoints follow the **Hexagonal Architecture** pattern:

1. **HTTP Handlers** (Inbound Adapter) - Handle HTTP requests/responses
2. **Application Services** - Orchestrate business logic
3. **Domain Layer** - Core business entities and rules
4. **Repositories** (Outbound Adapter) - Data persistence
5. **External Services** (Outbound Adapter) - AI, Redis, etc.

The flow is always:
```
Client → HTTP Handler → Application Service → Domain Logic → Repository/External Service
```
