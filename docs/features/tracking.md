# Features Tracking State

| Feature ID | Name | Status | Description |
| :--- | :--- | :--- | :--- |
| **feature01** | Core Telemetry & Processor MVP | `COMPLETED` | Real-time polling, CS/Min, GPM, and basic TUI dashboard with mock support. |
| **feature02** | Docker Compose Testing Environment | `COMPLETED` | Reproducible containerized environment for integration tests and CLI smoke tests against a mock Live Client Data API. |
| **feature03** | How to Generate a Windows Artifact | `COMPLETED` | Documentation and build helper for cross-compiling the Go CLI to a Windows executable. |
| **feature04** | Core Data Model Adherence (`allgamedata`) | `COMPLETED` | Expand SDK data models to match the full `allgamedata` JSON contract and add unmarshaling tests. |
| **feature05** | Live Client API Endpoint Abstractions | `COMPLETED` | Add individual Live Client Data API endpoint methods to the SDK and mock server. |
| **feature06** | Usage Examples and Overlay Documentation | `COMPLETED` | README examples and best practices for building a game overlay with the SDK. |
| **feature07** | Interactive CLI Monitor and Debugger | `COMPLETED` | Bubble Tea TUI with route navigation, raw JSON viewport, connection status, and resilient polling. |
| **feature08** | Interactive CLI Monitor and Debugger | `DUPLICATED` | Duplicate of feature07; kept for historical reference only. |
| **001-periodic-judge-hook** | Periodic Judge Hook | `READY_FOR_QA` | Trigger a Judge (LLM) hook at every 5-minute game mark, build a payload from Live Client Data API, and render short tactical advice in the TUI. |

---

*Gerado por OpenCode Agent | Tokens estimados: ~600 | Ciclos: 2*
