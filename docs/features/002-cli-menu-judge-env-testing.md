# 002-cli-menu-judge-env — Test Scenarios

> Quality planning for the CLI menu, BYOK Judge configuration, and in-terminal game tips.

## 1. Scope Validation

This document maps the acceptance criteria from the functional and technical specifications to concrete BDD/Gherkin scenarios. The implementation under test is composed of:

- `internal/judge/openrouter/client.go` — env-based model/key configuration.
- `internal/menu` — Bubble Tea menu model.
- `internal/tips` — Bubble Tea tips panel model.
- `cmd/lol-cli/main.go` — top-level wiring and message routing.

## 2. Happy Path Scenarios

### 2.1 Menu rendered on startup

```gherkin
Feature: CLI Feature Menu

  Scenario: Show menu when the CLI starts
    Given the user runs lol-cli.exe
    When the TUI initializes
    Then the menu shows "[Rotas do SDK]" and "[Dicas do Jogo]"
    And the cursor is on the first option
```

### 2.2 Navigate to SDK Routes

```gherkin
  Scenario: Select SDK Routes
    Given the menu is visible
    And the cursor is on "[Rotas do SDK]"
    When the user presses Enter
    Then the debugger view is rendered
    And the route list is visible
```

### 2.3 Navigate to Game Tips

```gherkin
  Scenario: Select Game Tips
    Given the menu is visible
    And the cursor is on "[Dicas do Jogo]"
    When the user presses Enter
    Then the tips panel is rendered
    And the current Judge configuration is shown
```

### 2.4 Default model when OPENROUTER_MODEL is absent

```gherkin
Feature: Judge BYOK Configuration

  Scenario: Use default model when env var is missing
    Given OPENROUTER_API_KEY is set
    And OPENROUTER_MODEL is not set
    When the Judge client is initialized
    Then the request model is "openai/gpt-4o-mini"
```

### 2.5 Custom model from OPENROUTER_MODEL

```gherkin
  Scenario: Use custom model when env var is present
    Given OPENROUTER_API_KEY is set
    And OPENROUTER_MODEL is "anthropic/claude-3.5-sonnet"
    When the Judge client is initialized
    Then the request model is "anthropic/claude-3.5-sonnet"
```

### 2.6 Judge disabled when API key is absent

```gherkin
  Scenario: Judge is disabled without API key
    Given OPENROUTER_API_KEY is not set
    When the CLI starts
    Then the Judge loop is not started
    And the tips panel shows "dicas desativadas"
```

## 3. Error Path Scenarios

### 3.1 Return to menu from SDK Routes

```gherkin
Feature: Navigation

  Scenario: Esc returns to menu from debugger
    Given the debugger view is active
    When the user presses Esc
    Then the menu is rendered again
```

### 3.2 Return to menu from Game Tips

```gherkin
  Scenario: Esc returns to menu from tips
    Given the tips panel is active
    When the user presses Esc
    Then the menu is rendered again
```

### 3.3 Quit from menu

```gherkin
  Scenario: Quit from menu
    Given the menu is visible
    When the user presses q
    Then the program exits
```

## 4. Edge Cases

### 4.1 Advice update while in Tips panel

```gherkin
Feature: Real-time Tips

  Scenario: New advice appears in the tips panel
    Given the tips panel is active
    And the Judge is configured
    When the orchestrator produces a new advice
    Then the tips panel displays the advice
    And the configuration panel remains visible
```

### 4.2 Advice update while in debugger

```gherkin
  Scenario: New advice appears in the debugger status bar
    Given the debugger view is active
    And the Judge is configured
    When the orchestrator produces a new advice
    Then the debugger status bar updates with the advice
```

### 4.3 API key masking

```gherkin
Feature: Security

  Scenario: API key is masked in the UI
    Given OPENROUTER_API_KEY is "sk-1234567890abcdef"
    When the tips panel renders the configuration
    Then the key is shown as "sk-...cdef"
```

### 4.4 Empty advice handling

```gherkin
  Scenario: Waiting message when no advice yet
    Given the tips panel is active
    And no advice has been received
    Then the panel shows "Aguardando dica do Judge..."
```

## 5. Unit Test Targets

| Component | Focus | Technique |
| :--- | :--- | :--- |
| `openrouter/client.go` | env var model override, default model | table-driven tests sem rede |
| `internal/menu` | cursor movement, selection message, quit | Bubble Tea messages in memória |
| `internal/tips` | config rendering, advice update, key masking | mock EnvConfig + Update messages |

## 6. Integration Test Targets

| Target | Setup | Validation |
| :--- | :--- | :--- |
| OpenRouter request body | `httptest` server | Confirma que `model` no JSON reflete `OPENROUTER_MODEL` |
| Menu -> debugger flow | `tea.Program` smoke | Menu emite `SelectMsg` e appModel ativa debugger |
| Menu -> tips flow | `tea.Program` smoke | Menu emite `SelectMsg` e appModel ativa tips |
| Advice routing | mock orchestrator | `AdviceMsg` atualiza tips e debugger ao mesmo tempo |

---

_Assinatura de consumo: QA Planejador — 1 ciclo, ~1.200 tokens estimados._
