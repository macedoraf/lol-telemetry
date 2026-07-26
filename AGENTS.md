# AGENTS.md

## 1. System State
- **Project:** `lol-telemetry` (Go CLI & SDK)
- **Objective:** Real-time LoL metrics extraction and tactical advice via the official Live Client Data API (Port 2999). Vanguard compliant (no memory reading).
- **Status:** Feature `002-cli-menu-judge-env` passed QA. Feature `001-periodic-judge-hook` is ready for QA.
- **Base Stack:** Go 1.22+, Charmbracelet Bubble Tea/Lipgloss, native `net/http`.

## 2. OpenCode Directives
- **Context Management:** This file and the contents of `.opencode/rules/` serve as the primary layer. Do NOT load full specification files into context unless executing the specific phase.
- **Strict Compliance:** Adhere strictly to the State Machine Workflow and Architecture Rules.

## 3. Index of Rules & Documentation
- **Rules (.opencode/rules/):**
  - Architecture & Boundaries: `.opencode/rules/architecture.md`
  - Guardrails: `.opencode/rules/guardrails.md`
- **Workflow Commands (.opencode/commands/):**
  - New feature: `.opencode/commands/workflow-new-feature.md`
  - Execute feature: `.opencode/commands/execute-feature.md`
- **Detailed Docs (docs/):**
  - Feature tracking: `docs/features/tracking.md`
  - Feature `001-periodic-judge-hook` specs: `docs/features/001-periodic-judge-hook-functional.md`, `docs/features/001-periodic-judge-hook-technical.md`, `docs/features/001-periodic-judge-hook-testing.md`
  - Feature `002-cli-menu-judge-env` specs: `docs/features/002-cli-menu-judge-env-functional.md`, `docs/features/002-cli-menu-judge-env-technical.md`, `docs/features/002-cli-menu-judge-env-testing.md`
  - API Contracts & Mocks: `docs/api-contract.md`
  - Testing Scenarios: `docs/testing.md`
  - Changelog: `CHANGELOG.md`