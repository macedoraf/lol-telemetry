# Architecture Specification (`architecture.md`)

## 1. Tech Stack
*   **Language:** Go (Golang) - Version 1.22+
*   **Binary:** Single, static, and portable (Windows, macOS, Linux).
*   **Networking:** Native `net/http` package with `http.Transport` configured to ignore self-signed certificates (`InsecureSkipVerify = true`).

## 2. Concurrency Pattern
*   Strict use of **Goroutines** and **Channels** for asynchronous communication between the `CollectorAgent` and the `RendererAgent`.
*   The main application thread manages the TUI lifecycle, while HTTP requests run in the background via a dedicated goroutine.

## 3. Graphical Interface (CLI / TUI)
*   **Framework:** Charmbracelet ecosystem.
    *   `Bubble Tea`: Elm-architecture-based model (Model-View-Update) for asynchronous state management.
    *   `Lip Gloss`: Modern CSS-like flexbox styling and formatting for the terminal.

## 4. Testing Quality & Unit Test Guidelines
To ensure unit tests remain lean, highly human-readable, and aligned with Go best practices:
*   **Table-Driven Tests:** Mandatory for all mathematical and parsing logic (e.g., calculations in `ProcessorAgent`). Use slices of structs containing `name`, `input`, and `expected` fields.
*   **No External Dependencies:** Unit tests must be pure and run entirely in-memory without network calls or external file system dependencies (use mock structures directly).
*   **Readability & Minimalism:** Test functions must be concise, descriptive, and strictly focused on a single behavior (Arrange-Act-Assert pattern).

## 5. Directory Structure & Import Rules (Standard Go Layout)

```text
lol-telemetry/
├── cmd/
│   └── lol-cli/
│       └── main.go          # Entrypoint. Initializes agents and channels. ZERO business logic.
├── internal/                # Private application code (inaccessible externally)
│   ├── processor/           # ProcessorAgent: Calculation logic (CS/Min, GPM).
│   ├── renderer/            # RendererAgent: Bubble Tea views and components.
│   └── types/               # Shared internal models and application states.
├── pkg/                     # Public and reusable SDK
│   └── riotclient/          # CollectorAgent: Agnostic HTTP client for the Live Client Data API.
├── testdata/                # Static files for tests and mocks
│   └── mocks/               # Real JSONs extracted from the Riot API (Contracts).
├── .opencode/
│   └── rules/               # Harness rules and OpenCode state machines.
└── docs/                    # Detailed documentation on-demand.

```

### 4.1 Strict Import Rules (Dependency Boundaries)

cmd/lol-cli CAN import internal/ and pkg/.

internal/ CAN import pkg/riotclient and internal/types/.

pkg/riotclient CANNOT import anything from internal/ or cmd/. It must remain 100% decoupled and independent of the CLI application.