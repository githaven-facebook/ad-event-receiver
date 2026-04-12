# API Documentation

## Endpoints

### POST /api/events
Submit a single event.

Request body:
```json
{
  "type": "click",
  "ad_id": "12345",
  "user_id": "67890"
}
```

### POST /api/events/bulk
Submit events in bulk (max 100).

### GET /api/health
Health check endpoint.

### GET /api/metrics
Prometheus metrics.

### POST /api/admin/flush
Force flush all pending Kafka messages.

### DELETE /api/admin/reset
Reset all internal counters and caches.
