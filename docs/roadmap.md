# Roadmap — Planned Features

> Source of truth for upcoming work. Detailed specs (functional/technical/testing) are produced per feature via the standard workflow before implementation.

## Priority Order

| Order | Feature | Name | Complexity | Depends On |
| :---: | :--- | :--- | :---: | :--- |
| 1 | F-003 | Judge Tip Language Selection | Low | — |
| 2 | F-004 | Runtime Trigger Configuration | Medium | — |
| 3 | F-005 | Telemetry Recording (Riot API Capture) | Medium | — |
| 4 | F-006 | Runtime Judge Prompt Editing | Medium | — |
| 5 | F-007 | Judge Tip Persistence & Telemetry Correlation | Medium | F-005 |
| 6 | F-008 | Time-Series Feature Engineering for Judge | High | F-005, F-007 |

---

## F-003 — Judge Tip Language Selection
- **Goal:** Select the language of Judge tactical tips.
- **Scope:** Supported languages at launch: `en`, `pt-BR`, `es`. Configurable via env var and/or CLI menu; Judge prompt instructs the LLM to answer in the selected language.
- **Notes:** Default `en`.

## F-004 — Runtime Trigger Configuration
- **Goal:** Choose at runtime which triggers fire Judge tips (replacing the fixed 5-minute hook).
- **Scope:**
  1. Daemon: expose trigger parameterization (e.g. periodic interval, event-based triggers).
  2. Test CLI: consume that parameterization and expose it in the terminal.
- **Notes:** Keep backward-compatible default (5-min periodic).

## F-005 — Telemetry Recording (Riot API Capture)
- **Goal:** Persist raw data fetched from the Live Client Data API for future use.
- **Scope:**
  - Parametrized opt-in flag (env/CLI) to enable capture.
  - Write snapshots as JSON (JSONL/vector files preferred for append efficiency).
  - **Must not block the main thread** — async writer (buffered channel + dedicated goroutine).
- **Notes:** File rotation/naming by game session.

## F-006 — Runtime Judge Prompt Editing
- **Goal:** Change the Judge prompt without redeploying.
- **Scope:**
  - Daemon: `PATCH/PUT` route for the prompt resource.
  - Test CLI: edit/apply prompt at runtime via that route.
- **Notes:** Persist override for the session; fallback to packaged default.

## F-007 — Judge Tip Persistence & Telemetry Correlation
- **Goal:** Store Judge tips to disk in a format that allows correlating each tip with the recorded Riot telemetry.
- **Scope:** Tip records include game time + correlation key (e.g. session ID + timestamp) matching F-005 recordings; efficient append-only format (JSONL).
- **Depends on:** F-005 (telemetry recording must exist to correlate against).

## F-008 — Time-Series Feature Engineering for Judge
- **Goal:** Structure Judge input as a time series of derived features (data-science style feature engineering over raw Riot API data), without modifying original layers/schemas.
- **Architecture:** Apply the **Transform pattern inside the context** — consumption logic stays separate from transformation logic; the Judge receives transformed features, not raw payloads.
- **Required analytical capabilities:**
  - Gold/min progression (self/team).
  - XP/min progression (self/team).
  - Gold/XP spike detection — own team.
  - Gold/XP spike detection — enemy team.
  - Player build analysis vs. matchup vs. enemy team composition vs. synergy with own team.
- **Documentation requirements (due to complexity):** document every extracted feature with:
  - Example raw input.
  - Transformation applied.
  - Justification (why the feature matters).
- **Final step:** include the transformed feature data in the file-recording pipeline (F-005/F-007).
- **Depends on:** F-005, F-007.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.200 | Ciclos: 1*
