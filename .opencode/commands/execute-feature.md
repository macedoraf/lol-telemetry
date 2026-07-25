# Command: /execute-feature

## Description
Instructs the OpenCode agent to pick up a pending feature, follow the workflow skill, execute implementation, and update the tracking ledger.

## Usage
`/execute-feature feature01`

## Agent Execution Instructions
1. Read `.opencode/skills/workflow-skill.md`.
2. Locate the requested feature ID in `docs/features/tracking.md`.
3. Execute the pipeline sequentially: Spec -> Tests -> Code -> Validation -> Status Update.