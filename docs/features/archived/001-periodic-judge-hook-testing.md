# 001-periodic-judge-hook — Test Scenarios

> Quality planning for the periodic Judge hook feature. Covers happy path, error handling, deduplication, and edge cases.

## 1. Scope Validation

This document maps the acceptance criteria from the functional and technical specifications to concrete BDD/Gherkin scenarios. The implementation under test is the pipeline composed of:

- `internal/hooks/periodic.go` — 5-minute absolute trigger
- `internal/hooks/registry.go` — hook registry
- `internal/judge/payload/builder.go` — payload builder
- `internal/judge/judge.go` — Judge orchestrator
- `internal/judge/openrouter/client.go` — OpenRouter client

## 2. Happy Path Scenarios

### 2.1 First trigger at 05:00

```gherkin
Feature: Periodic Judge Hook

  Scenario: Trigger exactly once at the first 5-minute mark
    Given the system detects an active League of Legends match
    And the hook registry contains the periodic 5-minute hook
    When the game clock reaches 05:00 for the first time
    Then the hook emits a single Trigger
    And the Judge is invoked exactly once
    And the response is rendered in the TUI
```

### 2.2 Subsequent 5-minute marks

```gherkin
  Scenario: Trigger at every subsequent 5-minute mark
    Given the system already processed the 05:00 mark
    When the game clock reaches 10:00
    Then the hook emits a single Trigger
    And the Judge is invoked exactly once

  Scenario: Trigger at 15:00 after 05:00 and 10:00
    Given the system already processed 05:00 and 10:00
    When the game clock reaches 15:00
    Then the hook emits a single Trigger
    And the Judge is invoked exactly once
```

### 2.3 Late start in the middle of a match

```gherkin
  Scenario: Start polling after a mark has already passed
    Given the system is started while the game clock is at 07:30
    When the game clock reaches 10:00
    Then the hook emits a single Trigger
    And the Judge is invoked exactly once
    And the 05:00 mark is not processed retroactively
```

### 2.4 Complete Judge payload

```gherkin
Feature: Judge Payload

  Scenario: Payload contains all required fields
    Given the periodic hook fired at 10:00
    When the payload builder receives the AllGameData snapshot
    Then the JudgeRequest contains:
      | field          | source                                         |
      | GameMinute     | 10                                             |
      | Matchup        | active player vs same-position enemy player    |
      | KDA            | active player KDA and enemy KDA when available |
      | Gold           | active player gold and enemy gold when available |
      | Items          | active player items and enemy items when available |
      | Objectives     | towers, dragons, barons, heralds by team       |
      | GameState      | score, alive/dead champions, relevant buffs    |
      | Question       | the hook's instruction                         |
      | SystemPrompt   | the judge system prompt                        |
```

## 3. Error Path Scenarios

### 3.1 No active match

```gherkin
Feature: Match Detection Guard

  Scenario: No trigger when the game clock is not positive
    Given the Live Client Data API returns gameTime <= 0
    When the hooks are evaluated
    Then the periodic hook does not emit a Trigger
    And the Judge is not invoked

  Scenario: No trigger when the API call fails
    Given the Live Client Data API returns an error
    When the hooks are evaluated
    Then the periodic hook does not emit a Trigger
    And the Judge is not invoked
```

### 3.2 Deduplication inside the same mark

```gherkin
Feature: Deduplication

  Scenario: Do not trigger repeatedly inside the same 5-minute window
    Given the system already processed the 10:00 mark
    When the game clock is polled multiple times between 10:00 and 14:59
    Then the periodic hook does not emit a Trigger
    And the Judge is not invoked again
```

### 3.3 LLM response too long

```gherkin
Feature: Judge Response Validation

  Scenario: Truncate or validate response over 140 characters
    Given the LLM returned a string with 200 characters
    When the Judge processes the response
    Then the JudgeResponse.Advice contains at most 140 characters
    And it remains a single actionable sentence
```

### 3.4 OpenRouter failure

```gherkin
Feature: OpenRouter Client Resilience

  Scenario: Handle OpenRouter timeout without crashing
    Given the OpenRouter call times out
    When the Judge executes
    Then the error is returned gracefully
    And the application does not panic

  Scenario: Handle OpenRouter 5xx error without crashing
    Given the OpenRouter returns HTTP 500
    When the Judge executes
    Then the error is returned gracefully
    And the application does not panic
```

### 3.5 Opponent not identified

```gherkin
Feature: Matchup Detection

  Scenario: Send request when opponent position is not found
    Given there is no enemy player with the same Position as the active player
    When the payload builder processes the snapshot
    Then the JudgeRequest is still sent
    And the Matchup field indicates "opponent not identified"
    And the other global fields are populated
```

## 4. Edge Cases

### 4.1 Multiple registered hooks

```gherkin
Feature: Hook Registry

  Scenario: Multiple hooks trigger independently
    Given two distinct hooks are registered in the registry
    When both hooks fire in the same polling cycle
    Then each hook generates its own JudgeRequest
    And the requests do not interfere with each other
```

### 4.2 Reset between matches

```gherkin
Feature: State Reset Between Matches

  Scenario: 5-minute marks are reprocessed in a new match
    Given the previous match ended after the 10:00 mark was processed
    And a new active match is detected
    When the new game clock reaches 05:00
    Then the periodic hook emits a Trigger
    And the Judge is invoked for the 05:00 mark of the new match
```

> **Confirmed behavior:** the `Orchestrator` resets `HookContext.PrevFired` whenever the system detects the end of a match (`gameTime <= 0`) or a failure in the Live Client Data API call. This guarantees that the 5-minute marks of a new match are reprocessed from the beginning.

## 5. Unit Test Targets

| Component | Focus | Technique |
| :--- | :--- | :--- |
| `periodic.go` | mark calculation, deduplication, late start | table-driven tests with gameTime from 0 to 1200 seconds |
| `payload/builder.go` | field extraction, matchup identification, enemy fallback | in-memory mocks of `AllGameData` |
| `judge.go` | system prompt construction, response truncation | mock `LLMClient` |
| `openrouter/client.go` | HTTP headers, JSON body, response parsing | `httptest` server |
| `registry.go` | registration and independent execution | simple Go tests |

## 6. Integration Test Targets

| Target | Setup | Validation |
| :--- | :--- | :--- |
| End-to-end orchestrator | `httptest` for Live Client Data API + mock LLMClient | Verify hook fires at 05:00 and payload reaches the Judge |
| OpenRouter client | `httptest` simulating OpenRouter `/v1/chat/completions` | Verify `Authorization` and `Content-Type` headers, JSON body, and response parsing |

---

*Gerado por QA Planejador | Tokens estimados: ~1.500 | Ciclos: 1*
