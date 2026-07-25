# Testing Scenarios & Strategy (`testing.md`)

## 1. Manual & Mock Mode Scenarios (Offline / Development)
To allow visual validation of the TUI (Charmbracelet Bubble Tea) without an active LoL game client, the CLI supports a mock execution mode[cite: 6].

*   **Scenario 1.1 (Success - Mock CLI Boot):** 
    *   *Command:* `go run cmd/lol-cli/main.go --mock testdata/mocks/allgamedata.json`[cite: 6]
    *   *Expected Behavior:* The TUI starts successfully, reads the local JSON file in a loop (polling simulation), and renders active player stats and metrics smoothly[cite: 6].
*   **Scenario 1.2 (Failure - Missing or Corrupted Mock File):**
    *   *Command:* `go run cmd/lol-cli/main.go --mock testdata/mocks/invalid.json`[cite: 6]
    *   *Expected Behavior:* The application catches the file read/parsing error gracefully, logs a clear error message to stderr without panicking, and exits with a non-zero status code[cite: 6].

## 2. Integration Scenarios (Live Client Real Integration)
Scenarios to validate behavior when interacting with a real running instance of League of Legends on port `2999`[cite: 6].

*   **Scenario 2.1 (Success - Active Match Polling):**
    *   *Condition:* LoL client is running a custom or training match[cite: 6].
    *   *Command:* `go run cmd/lol-cli/main.go` (without `--mock`)[cite: 6]
    *   *Expected Behavior:* The HTTP client successfully bypasses self-signed SSL, polls `https://127.0.0.1:2999/liveclientdata/allgamedata`, parses the live payload, and updates the TUI in real-time[cite: 6].
*   **Scenario 2.2 (Failure - Client Closed / Network Timeout):**
    *   *Condition:* LoL client is closed or not in an active match[cite: 6].
    *   *Command:* `go run cmd/lol-cli/main.go`[cite: 6]
    *   *Expected Behavior:* The `CollectorAgent` handles connection refusal/timeouts gracefully, displaying a status message ("Waiting for League of Legends match...") and retrying periodically without crashing[cite: 6].

## 3. Automation Test Scenarios (Integration Tests with httptest)
*   **Scenario 3.1 (HTTP Client Mock Server):**
    *   Use Go's `net/http/httptest` to spin up a local test server responding with valid and invalid payloads[cite: 6].
    *   Verify that `riotclient.GetGameData()` parses success responses correctly and returns typed errors on HTTP 500 or malformed JSON[cite: 6].

## 4. Containerized Integration Scenarios (Docker Compose)
To validate the full stack against a mock Live Client Data API without a local Go toolchain or a running LoL client.

*   **Scenario 4.1 (Success - Docker Compose Full Suite):**
    *   *Command:* `docker compose up --build`.
    *   *Expected Behavior:* The `mock-api` service becomes healthy on its `/health` endpoint, the `tests` service runs `go test ./...`, the `integration` tagged test in `pkg/riotclient` successfully fetches `https://mock-api:2999/liveclientdata/allgamedata`, and the CLI smoke test with `--mock testdata/mocks/allgamedata.json --smoke-test` completes without error. The compose command exits with status `0`.
*   **Scenario 4.2 (Failure - Mock API Unhealthy):**
    *   *Condition:* The `mock-api` service fails to start or its health check never returns HTTP 200.
    *   *Expected Behavior:* Docker Compose reports the `mock-api` service as unhealthy and the `tests` service never starts, surfacing the failure clearly.

## 5. Unit Test Scenarios (TDD Math Logic)
*   **Scenario 5.1 (CS/Min Calculation):**
    *   *Input:* 120 total minions/monsters killed, game time = 600 seconds (10 minutes)[cite: 6].
    *   *Expected Output:* `12.0` CS/Min[cite: 6].
*   **Scenario 5.2 (GPM - Gold Per Minute Calculation):**
    *   *Input:* 3000 total gold, game time = 300 seconds (5 minutes)[cite: 6].
    *   *Expected Output:* `600.0` Gold Per Minute[cite: 6].