# lol-telemetry

A real-time telemetry SDK and CLI for League of Legends, built in Go. Fully compliant with Riot Games Vanguard anti-cheat policies (zero memory reading).

## 🚀 Quickstart

### Prerequisites
* Go 1.22+ installed.

### Running in Mock Mode (Offline Development)
Test the TUI dashboard instantly without an active game client:
```bash
go run cmd/lol-cli/main.go --mock testdata/mocks/allgamedata.json

## Testing with Docker Compose

A containerized test stack is provided for reproducible builds and integration tests without installing a local Go toolchain or running the League of Legends client.

### Prerequisites
* Docker and Docker Compose installed.

### Run the Full Test Suite
```bash
docker compose up --build
```

This command:
1. Builds the Go test runner image.
2. Builds and starts the mock Live Client Data API on `https://localhost:2999`.
3. Runs unit tests (`go test ./...`).
4. Runs the integration test against the mock API.
5. Runs the CLI smoke test in mock mode (`--mock --smoke-test`) without requiring an interactive terminal.

### Start the Mock API for Ad-Hoc Testing
```bash
docker compose up mock-api -d
```

The mock API exposes:
* `https://localhost:2999/liveclientdata/allgamedata` (self-signed certificate)
* `http://localhost:8080/health` (health check endpoint)

You can point the `riotclient` integration tests at it by setting `MOCK_API_URL=https://mock-api:2999/liveclientdata` from within the compose network.