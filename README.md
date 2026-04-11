# ad-event-receiver

A high-throughput HTTP service that receives, classifies, and routes ad events to Kafka topics for downstream processing by SSP and DSP pipelines.

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │         ad-event-receiver            │
                    │                                      │
HTTP Client ──────► │  POST /api/v1/events                 │
                    │  POST /api/v1/events/batch           │
                    │                                      │
                    │  ┌────────────┐   ┌───────────────┐  │
                    │  │  Handler   │──►│ Kafka Producer │  │──► ad-events-ssp
                    │  │  (chi)     │   │ (kafka-go)    │  │
                    │  └────────────┘   └───────────────┘  │──► ad-events-dsp
                    │                                      │
                    │  GET /healthz  GET /readyz            │
                    │  GET /metrics (port 9090)            │
                    └─────────────────────────────────────┘
```

Events are classified as **SSP** (Supply-Side Platform / publisher side) or **DSP** (Demand-Side Platform / advertiser side) based on the `event_type` field and routed to separate Kafka topics. Downstream consumers can then independently process inventory and buyer-side data streams.

## API

### POST /api/v1/events

Ingest a single ad event.

**Request**
```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "SSP",
  "timestamp": "2024-03-15T10:30:00Z",
  "payload": {
    "ad_id": "ad-001",
    "campaign_id": "campaign-001",
    "creative_id": "creative-001",
    "publisher_id": "pub-001",
    "user_id": "user-hash-abc123",
    "action": "impression",
    "amount": 0.05,
    "metadata": {
      "placement": "top_banner",
      "format": "300x250"
    }
  }
}
```

**Response (202 Accepted)**
```json
{
  "status": "ok",
  "message": "event accepted",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "abc123",
  "timestamp": "2024-03-15T10:30:00.123Z"
}
```

**Event Types**

| Field | Values |
|-------|--------|
| `event_type` | `SSP`, `DSP` |
| `payload.action` | `impression`, `click`, `conversion`, `view_through` |

### POST /api/v1/events/batch

Ingest up to 1,000 events in a single request.

**Request**
```json
{
  "events": [
    { ... },
    { ... }
  ]
}
```

**Response (202 Accepted)**
```json
{
  "status": "ok",
  "message": "batch accepted",
  "accepted": 2,
  "request_id": "xyz789",
  "timestamp": "2024-03-15T10:30:00.456Z"
}
```

### GET /healthz

Liveness probe. Returns `200 OK` while the process is running.

### GET /readyz

Readiness probe. Returns `200 OK` when all dependencies are healthy, `503` otherwise.

### GET /metrics (port 9090)

Prometheus metrics endpoint.

| Metric | Type | Description |
|--------|------|-------------|
| `ad_event_receiver_events_received_total` | Counter | Events received, by `event_type` |
| `ad_event_receiver_events_published_total` | Counter | Events published to Kafka, by `topic` |
| `ad_event_receiver_event_processing_duration_seconds` | Histogram | Processing latency, by `event_type` |
| `ad_event_receiver_kafka_publish_errors_total` | Counter | Kafka publish failures, by `topic` |
| `ad_event_receiver_http_requests_total` | Counter | HTTP requests, by `method`, `path`, `status` |
| `ad_event_receiver_http_request_duration_seconds` | Histogram | HTTP latency, by `method`, `path` |

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP server port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated broker addresses |
| `KAFKA_SSP_TOPIC` | `ad-events-ssp` | Kafka topic for SSP events |
| `KAFKA_DSP_TOPIC` | `ad-events-dsp` | Kafka topic for DSP events |
| `KAFKA_BATCH_SIZE` | `100` | Kafka producer batch size |
| `KAFKA_WRITE_TIMEOUT` | `10s` | Kafka write timeout |
| `KAFKA_REQUIRED_ACKS` | `1` | Required Kafka acks (0=none, 1=leader, -1=all) |
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `LOG_FORMAT` | `json` | Log format: json or console |
| `SERVER_READ_TIMEOUT` | `10s` | HTTP read timeout |
| `SERVER_WRITE_TIMEOUT` | `30s` | HTTP write timeout |
| `SERVER_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown timeout |

## Local Development

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- [golangci-lint](https://golangci-lint.run/) v1.57+

### Run with Docker Compose

```bash
# Start all services (app + Kafka + Zookeeper)
make docker-run

# Check logs
docker-compose logs -f ad-event-receiver

# Send a test event
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_type": "SSP",
    "timestamp": "2024-03-15T10:30:00Z",
    "payload": {
      "ad_id": "ad-001",
      "campaign_id": "campaign-001",
      "action": "impression",
      "amount": 0.05
    }
  }'
```

### Run locally

```bash
# Start only Kafka and Zookeeper
docker-compose up -d zookeeper kafka

# Run the service
make run

# Or with custom config
SERVER_PORT=8080 KAFKA_BROKERS=localhost:29092 make run
```

### Tests

```bash
# Unit tests
make test

# Lint
make lint

# Integration tests (requires running Kafka)
./scripts/run-integration-tests.sh
```

## Deployment

### Building the Docker image

```bash
make docker-build
# Or with a specific tag
docker build -t ad-event-receiver:v1.2.3 .
```

### Kubernetes

The service exposes:
- Port `8080` for the main HTTP API
- Port `9090` for Prometheus metrics

Configure your Kubernetes deployment with:
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### CI/CD

- **CI** runs on every push/PR: lint → unit tests → build → docker push (main only)
- **CD** triggers on version tags (`v*.*.*`): build → staging deploy → production deploy

## Project Structure

```
ad-event-receiver/
├── cmd/receiver/          # Main entrypoint
├── internal/
│   ├── config/            # Environment-based configuration
│   ├── handler/           # HTTP handlers and middleware
│   ├── kafka/             # Kafka producer interface and implementation
│   ├── metrics/           # Prometheus metrics definitions
│   └── server/            # Router and HTTP server lifecycle
├── test/integration/      # Integration tests (build tag: integration)
├── scripts/               # Helper scripts
├── Dockerfile             # Multi-stage distroless build
└── docker-compose.yml     # Local development environment
```
