# Ad Event Receiver - Claude Code Instructions

## Project Overview
Go HTTP server that receives advertising events via REST API and publishes them to Kafka topics, classified by SSP (Supply-Side) and DSP (Demand-Side) event types.

## Tech Stack
- Go 1.22
- HTTP framework: chi router
- Kafka: segmentio/kafka-go
- Logging: zap
- Metrics: prometheus/client_golang
- Config: environment variables

## Directory Structure
- `cmd/receiver/` - Application entry point
- `internal/config/` - Configuration loading
- `internal/handler/` - HTTP request handlers
- `internal/kafka/` - Kafka producer implementation
- `internal/metrics/` - Prometheus metrics
- `internal/model/` - Domain models (events, responses)
- `internal/server/` - HTTP server and router setup
- `test/integration/` - Integration tests

## Commands
- Build: `make build`
- Test: `make test`
- Lint: `make lint` (requires golangci-lint)
- Run locally: `make run`
- Docker: `make docker-build && make docker-run`
- Integration tests: `./scripts/run-integration-tests.sh`

## Coding Standards
- Follow standard Go project layout
- All exported functions must have godoc comments
- Use `context.Context` as first parameter in all functions
- Error wrapping with `fmt.Errorf("...: %w", err)`
- Table-driven tests for all handler and service functions
- Interfaces defined in consumer package, not producer

## Kafka Topics
- SSP events → `ad.events.ssp` topic
- DSP events → `ad.events.dsp` topic
- Dead letter → `ad.events.dlq` topic

## Do NOT
- Do not modify Kafka topic names without updating ad-event-stream consumers
- Do not change the event schema (model/event.go) without versioning
- Do not bypass the middleware chain in router.go
- Do not use global variables for state

## Dependencies on Other Services
- Downstream: ad-event-stream (Flink) consumes from Kafka topics
- This service is stateless; all state is in Kafka
