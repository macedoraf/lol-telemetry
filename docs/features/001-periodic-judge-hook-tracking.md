# Features Tracking State

| Feature ID | Name | Status | Description |
| :--- | :--- | :--- | :--- |
| **001-periodic-judge-hook** | Periodic Judge Hook | `READY_FOR_QA` | Trigger a Judge (LLM) hook at every 5-minute game mark, build a payload from Live Client Data API, and render short tactical advice in the TUI. |

## Artifacts

| Artifact | Path | Status |
| :--- | :--- | :--- |
| Functional specification | `docs/features/001-periodic-judge-hook-functional.md` | `APPROVED` |
| Technical specification | `docs/features/001-periodic-judge-hook-technical.md` | `APPROVED` |
| Test scenarios | `docs/features/001-periodic-judge-hook-testing.md` | `APPROVED` |
| Implementation | `internal/hooks/`, `internal/judge/`, `internal/orchestrator/`, `internal/types/`, `internal/renderer/`, `cmd/lol-cli/main.go` | `DONE` |
| Defect fix | `retroactive-first-mark`: cold-start baseline in `internal/hooks/periodic.go` and `internal/orchestrator/orchestrator.go` | `FIXED` |

## Next Step

QA review: run the full test suite and validate acceptance criteria before marking `COMPLETED`.

---

*Gerado por OpenCode Agent | Tokens estimados: ~400 | Ciclos: 1*
