# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`002-cli-menu-judge-env`**: Interactive CLI menu and BYOK Judge configuration.
  - `OPENROUTER_MODEL` environment variable to configure the LLM model used by the Judge (defaults to `openai/gpt-4o-mini`).
  - Feature menu (`internal/menu`) on startup with `[Rotas do SDK]` and `[Dicas do Jogo]` options.
  - Tips panel (`internal/tips`) that displays the latest Judge advice and current environment configuration (API key is masked in the UI).
  - Top-level `appModel` in `cmd/lol-cli/main.go` that routes between menu, debugger, and tips while forwarding Judge advice to all views.

## [0.3.0] - 2026-07-25

### Added

- **`001-periodic-judge-hook`**: Periodic LLM-driven tactical advice for the CLI.
  - Hook registry (`internal/hooks`) with a 5-minute absolute trigger and deduplication.
  - Judge orchestrator (`internal/judge`) that builds a structured payload from the Live Client Data API and calls an OpenRouter-compatible LLM.
  - Response truncation to a single actionable sentence of at most 140 characters.
  - Orchestrator state reset between matches so past marks are not reprocessed in a new game.
  - TUI rendering of Judge advice via Bubble Tea.

## [0.2.0] - 2026-07-24

### Added

- `feature07`: Interactive CLI monitor and debugger with Bubble Tea TUI.
- `feature06`: Overlay examples and usage documentation.
- `feature05`: Individual Live Client Data API endpoint abstractions.
- `feature04`: Core data model adherence to the full `allgamedata` contract.
- `feature03`: Windows artifact build helper and documentation.
- `feature02`: Docker Compose testing environment with mock API.
- `feature01`: Core telemetry and processor MVP (CS/Min, GPM, basic TUI, mock mode).

[Unreleased]: https://github.com/rafael-macedo/lol-telemetry/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/rafael-macedo/lol-telemetry/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/rafael-macedo/lol-telemetry/releases/tag/v0.2.0

---

*Gerado por Tech Writer | Tokens estimados: ~1.200 | Ciclos: 2*
