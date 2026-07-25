# Feature 01 Specification: Core Telemetry & Processor MVP

## 1. Objective
Implement the foundational capability to fetch data from the local Riot Live Client Data API, process core real-time metrics (CS/Min and GPM), and display them via a TUI dashboard.

## 2. Requirements
- **Collector:** Implement HTTP client in `pkg/riotclient` to fetch `/liveclientdata/allgamedata` with SSL bypass (`InsecureSkipVerify = true`).
- **Processor:** Implement calculation logic in `internal/processor` for:
  - **CS/Min:** Total minions and monsters killed divided by game time in minutes.
  - **GPM:** Total current gold divided by game time in minutes.
- **Renderer:** Implement a basic Bubble Tea TUI dashboard in `internal/renderer` displaying active player stats.
- **Mock Support:** Add a `--mock` flag to CLI (`cmd/lol-cli/main.go`) to read local JSON payloads from `testdata/mocks/allgamedata.json`.