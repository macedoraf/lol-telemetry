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
| **002-cli-menu-judge-env** | Interactive CLI Menu & BYOK Judge Config | `QA_PASSED` | BYOK Judge configuration via environment variables and an interactive CLI menu to access SDK routes or game tips. |
| **003-judge-tip-language** | Judge Tip Language Selection | `IMPLEMENTED` | Select Judge tip language (`en`, `pt-BR`, `es`) via `JUDGE_LANGUAGE`; runtime override seam for F-004. |
| **004-runtime-trigger-config** | Runtime Trigger Configuration | `IMPLEMENTED` | `GET/PATCH /api/config` on the daemon: enable/disable hooks, tune hook params and language at runtime; new Config tab in the test CLI. |
| **005-telemetry-recording** | Telemetry Recording | `IMPLEMENTED` | Opt-in async JSONL recording of raw Live Client Data API snapshots per game session; never blocks the poll loop. |
| **006-runtime-prompt-editing** | Runtime Judge Prompt Editing | `IMPLEMENTED` | PATCH judge system prompt at runtime via `/api/config`; view/edit from the CLI via `$EDITOR`. |
| **007-judge-tip-persistence** | Judge Tip Persistence & Correlation | `IMPLEMENTED` | Record Judge tips to `tips.jsonl` correlated with telemetry via `(session, gameTime)`. |
| **008-timeseries-feature-engineering** | Time-Series Feature Engineering for Judge | `IMPLEMENTED` | Transform pipeline over raw API data: gold/min, XP/min (level-derived), team/enemy spikes, objectives/death timers via enriched events, matchup diffs; additive `Features` in JudgeRequest; `features.jsonl` recording. |

---

*Gerado por OpenCode Agent | Tokens estimados: ~800 | Ciclos: 3*
