# Feature 02 Specification: Docker Compose Testing Environment

## 1. Objective
Provide a reproducible, containerized testing environment using Docker Compose so that integration tests and CLI smoke tests can run against a mock Live Client Data API without requiring a running League of Legends client or a local Go toolchain.

## 2. Requirements

### 2.1 Docker Compose Stack (`docker-compose.yml`)
- **Service `mock-api`**
  - Lightweight container that exposes an HTTPS server on port `2999`.
  - Serves a valid `allgamedata` JSON payload identical in structure to `testdata/mocks/allgamedata.json`.
  - Uses a self-signed TLS certificate so the `riotclient.Client` can connect with `InsecureSkipVerify = true`, matching the behavior of the real LoL client.
- **Service `tests`**
  - Go 1.22+ container.
  - Mounts the project source at build/runtime.
  - Runs the full test suite (`go test ./...`).
  - Runs the CLI in mock mode as a smoke test (`go run cmd/lol-cli/main.go --mock testdata/mocks/allgamedata.json --smoke-test`). The `--smoke-test` flag executes a single fetch/calculate cycle and exits without the interactive Bubble Tea TUI, making it suitable for CI containers.
  - Depends on `mock-api` being healthy before test execution.

### 2.2 Mock Server Configuration
- Mock server image should be based on a minimal base (e.g., `python:3.12-alpine`, `nginx:alpine`, or a tiny Go binary).
- Endpoint: `https://mock-api:2999/liveclientdata/allgamedata`.
- Must return HTTP 200 with a JSON body that validates against `pkg/riotclient.AllGameData`.
- Configurable payload via volume mount so the same compose file can serve both `allgamedata.json` and `invalid.json` for negative testing.

### 2.3 Build & Test Commands
- `docker compose up --build` should:
  1. Build/refresh the Go test image.
  2. Start the mock API.
  3. Run all Go unit/integration tests.
  4. Run the CLI smoke test.
  5. Exit with the test container's exit code.
- `docker compose up mock-api -d` should allow local/CI ad-hoc testing against the mock server.

### 2.4 Healthchecks & Networking
- Define a `healthcheck` for `mock-api` so the `tests` service waits until the API is ready. The mock server exposes a dedicated HTTP `/health` endpoint on port `8080` for health checks so the `tests` service does not depend on TLS certificate validation.
- Use an explicit Docker bridge network so the `tests` container can resolve `mock-api` by service name.

### 2.5 Documentation Updates
- Add a "Testing with Docker Compose" section to `README.md`.
- Add a new scenario to `docs/testing.md` describing the containerized integration test.
- Keep existing manual and `httptest` scenarios intact.

## 3. Out of Scope
- Multi-stage production image for the CLI.
- Publishing images to a registry.
- Running the Bubble Tea TUI interactively inside the `tests` container (only smoke test with `--mock` and a short timeout).

## 4. Acceptance Criteria
- [x] `docker compose up --build` exits `0` on a clean repository.
- [x] Unit tests (`pkg/riotclient`, `internal/processor`, `internal/renderer`) pass inside the container.
- [x] CLI smoke test with `--mock` completes without error.
- [x] The `tests` service waits for the mock API to be healthy before executing.
- [x] `README.md` contains the new Docker Compose testing instructions.
- [x] `docs/testing.md` contains a new containerized integration scenario.
