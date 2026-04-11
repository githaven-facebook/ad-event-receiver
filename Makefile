BINARY_NAME=ad-event-receiver
BUILD_DIR=bin
CMD_DIR=cmd/receiver
IMAGE_NAME=ad-event-receiver
IMAGE_TAG?=latest

.PHONY: all build test lint run docker-build docker-run clean help

all: lint test build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/...
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Test coverage report: coverage.html"

lint:
	@echo "Running linters..."
	golangci-lint run ./...

run:
	@echo "Starting $(BINARY_NAME)..."
	go run ./$(CMD_DIR)/...

docker-build:
	@echo "Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-run:
	@echo "Running Docker container..."
	docker-compose up -d

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests with coverage"
	@echo "  lint         - Run golangci-lint"
	@echo "  run          - Run the service locally"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Start services with docker-compose"
	@echo "  clean        - Remove build artifacts"
