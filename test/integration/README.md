# Integration Tests

These tests exercise the full event flow from HTTP ingestion through Kafka publication
and verify that events land on the correct topics.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.22+

## Running Integration Tests

### Option 1: Using the helper script (recommended)

```bash
./scripts/run-integration-tests.sh
```

This script starts the required services, waits for them to be healthy, runs the
integration tests, and tears down the environment.

### Option 2: Manual

1. Start Kafka and Zookeeper:

```bash
docker-compose up -d zookeeper kafka
```

2. Wait for Kafka to be healthy:

```bash
docker-compose ps
```

3. Run the tests with the `integration` build tag:

```bash
go test -v -tags=integration -timeout=120s ./test/integration/...
```

4. Tear down:

```bash
docker-compose down
```

## Test Coverage

| Test | Description |
|------|-------------|
| `TestEventFlow_SSP` | Posts a single SSP event and verifies it lands on the SSP Kafka topic |
| `TestEventFlow_DSP` | Posts a single DSP event and verifies it lands on the DSP Kafka topic |
| `TestBatchEventFlow` | Posts a batch of mixed events and verifies all are accepted |

## Configuration

The integration tests use the following defaults (matching `docker-compose.yml`):

| Variable | Default |
|----------|---------|
| Kafka broker | `localhost:29092` |
| SSP topic | `test-ad-events-ssp` |
| DSP topic | `test-ad-events-dsp` |

Override by editing the constants at the top of `kafka_integration_test.go`.
