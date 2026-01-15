# Configuration Guide

## Environment-Based Configuration

The AI Smart Queue uses environment-specific configuration files:

- `config.dev.yaml` - Development/staging configuration (simulation enabled) - **DEFAULT**
- `config.prod.yaml` - Production configuration (simulation disabled)

## Using Different Configurations

### Via Environment Variable

Set the `CONFIG_ENV` environment variable to switch configurations:

```bash
# Development (with failure simulation)
export CONFIG_ENV=dev
docker-compose up -d

# Production (no simulation)
export CONFIG_ENV=prod
docker-compose up -d
```

### Docker Compose

The default is `dev`:

```bash
# Uses config.dev.yaml
docker-compose up -d

# Override to production
CONFIG_ENV=prod docker-compose up -d
```

### Vercel/Cloud Deployment

Set the environment variable in your deployment platform:

```
CONFIG_ENV=prod
```

## Failure Simulation

### Configuration

In `config.dev.yaml`:

```yaml
simulation:
  enabled: true
  failure_rate: 0.3  # 30% of jobs will fail randomly
```

In `config.prod.yaml`:

```yaml
simulation:
  enabled: false
  failure_rate: 0.0
```

### How It Works

When simulation is enabled:

1. **Random Failures**: Jobs fail randomly based on the `failure_rate` (0.0 to 1.0)
2. **Realistic Errors**: Each job type has realistic error messages:
   - **Email**: SMTP errors, DNS failures, mailbox full, etc.
   - **Notification**: Push service unavailable, rate limits, invalid tokens, etc.
   - **Data Processing**: Memory errors, JSON parsing, database issues, etc.
3. **No Payload Flag Needed**: Simulation is config-driven, not payload-driven
4. **All Job Types**: Works for all job types (email, notification, data_processing)

### Example Error Messages

**Email Jobs:**
- "failed to connect to SMTP server: connection timeout"
- "SMTP authentication failed: invalid credentials"
- "email rejected by recipient server: mailbox full"
- "DNS lookup failed for mail server"

**Notification Jobs:**
- "push notification service unavailable"
- "invalid device token"
- "rate limit exceeded for notifications"

**Data Processing Jobs:**
- "out of memory during data processing"
- "invalid data format: JSON parsing error"
- "database connection lost during transaction"

## AI Insights Configuration

### Local vs Remote Insights

**Development (config.dev.yaml)**:
```yaml
ai:
  ollama_url: "http://ollama:11434"
  model: "phi3:mini"              # 2.3GB, optimized for ARM
  insights_url: ""                # Empty = use local Ollama
```

**Production (config.prod.yaml)**:
```yaml
ai:
  ollama_url: "http://ollama:11434"
  model: "phi3:mini"
  insights_url: "http://163.176.243.66:8082"  # Remote insights API on VM1
```

### How It Works

- **insights_url empty**: Worker uses local Ollama service directly
- **insights_url set**: Worker calls remote insights API via HTTP (5-min timeout)
- Cache check via `GetByJobID` prevents redundant AI analysis

## Testing

Create jobs normally without any special payload flags:

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "default",
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Test"
    }
  }'
```

With `simulation.enabled: true` and `failure_rate: 0.3`, approximately 30% of jobs will fail with realistic error messages.
