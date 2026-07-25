# AGENTS.md

## Overview
Project: `lol-telemetry` (Go CLI & SDK)
Purpose: Real-time LoL metrics extraction via Live Client Data API (Port 2999). Vanguard compliant (no memory reading).

## OpenCode Directives
- **Context Management:** This file and the contents of `.opencode/rules/` serve as the primary layer. Do NOT load full specification files into context unless executing the specific phase.
- **Strict Compliance:** Adhere strictly to the State Machine Workflow and Architecture Rules.

## Index of Rules & Documentation
- **Rules (.opencode/rules/):**
  - Architecture & Boundaries: `.opencode/rules/architecture.md`
  - Development Workflow (SDLC): `.opencode/rules/workflow.md`
- **Detailed Docs (docs/):**
  - Specs (Tech & Functional): `docs/specs.md`
  - API Contracts & Mocks: `docs/api-contract.md`
  - Testing Scenarios: `docs/testing.md`