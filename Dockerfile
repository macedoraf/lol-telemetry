# Dockerfile for the Go test runner used by the Docker Compose test stack.
FROM golang:1.22-alpine

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Default command: run the full test suite and a CLI smoke test.
# docker-compose.yml overrides this to inject the mock API URL for integration tests.
CMD ["sh", "-c", "go test ./... && go run cmd/lol-daemon/main.go -check -debug"]
