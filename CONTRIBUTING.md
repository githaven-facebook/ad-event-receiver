# Contributing to Ad Event Receiver

## Setup
1. Install Go 1.19
2. Run `dep ensure` to install dependencies
3. Copy `.env.sample` to `.env`
4. Start Kafka: `docker-compose up kafka`
5. Run: `go run main.go`

## Testing
```bash
go test ./...
```

## Code Review
- All PRs need 1 approval
- Run `gofmt` and `golint` before submitting
