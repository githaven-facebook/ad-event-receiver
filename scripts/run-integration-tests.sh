#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

cleanup() {
  echo "==> Tearing down docker-compose services..."
  docker-compose down --remove-orphans
}
trap cleanup EXIT

echo "==> Starting Kafka and Zookeeper..."
docker-compose up -d zookeeper kafka

echo "==> Waiting for Kafka to become healthy..."
MAX_RETRIES=30
RETRY_INTERVAL=5
for i in $(seq 1 $MAX_RETRIES); do
  STATUS=$(docker-compose ps kafka --format json 2>/dev/null | grep -o '"Health":"[^"]*"' | cut -d'"' -f4 || echo "")
  if [[ "$STATUS" == "healthy" ]]; then
    echo "    Kafka is healthy after $((i * RETRY_INTERVAL))s"
    break
  fi
  if [[ $i -eq $MAX_RETRIES ]]; then
    echo "ERROR: Kafka did not become healthy within $((MAX_RETRIES * RETRY_INTERVAL))s"
    docker-compose logs kafka
    exit 1
  fi
  echo "    Waiting... ($i/$MAX_RETRIES)"
  sleep $RETRY_INTERVAL
done

echo "==> Creating Kafka test topics..."
docker-compose exec -T kafka kafka-topics \
  --create --if-not-exists \
  --bootstrap-server localhost:9092 \
  --replication-factor 1 \
  --partitions 3 \
  --topic test-ad-events-ssp

docker-compose exec -T kafka kafka-topics \
  --create --if-not-exists \
  --bootstrap-server localhost:9092 \
  --replication-factor 1 \
  --partitions 3 \
  --topic test-ad-events-dsp

echo "==> Running integration tests..."
go test \
  -v \
  -tags=integration \
  -timeout=120s \
  -count=1 \
  ./test/integration/...

echo "==> Integration tests passed."
