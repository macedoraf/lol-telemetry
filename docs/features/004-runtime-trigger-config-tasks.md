# 004-runtime-trigger-config — Tasks de Implementação

## 1. Objetivo

Parametrizar os triggers do Judge em runtime: daemon expõe `GET /api/config` e `PATCH /api/config` (mesmo servidor HTTP do `/ws`); o CLI de testes (`lol-cli`) ganha a tab **Config** listando hooks com enable/disable e edição de parâmetros, além de troca de idioma (reuso do seam da F-003).

## 2. Estado Atual

- `internal/hooks/periodic.go` — interface `Hook` + `Registry` (sem enable/disable, sem params). `Periodic5MinHook` tem intervalo fixo de 300s (`CurrentMark`).
- `internal/hooks/event.go` — 7 hooks com constantes hardcoded: `RecallHook` (1000g, 60s), `LaningPhaseEndHook` (840s).
- `internal/orchestrator/orchestrator.go` — `prevFired map[string]int64` privado; sem reset por hook.
- `pkg/service/daemon.go` — mux com `/ws` e `/health` apenas; `builder` é variável local de `NewDaemon`.
- `internal/tui/model.go` — tabs Live/Advice/Events/Log (`tabCount = 4`); CLI fala apenas WebSocket.
- F-003 entregou `Builder.SetLanguage` — este feature o consome.

## 3. Design

- **Contrato HTTP** (DTOs em `pkg/service/types.go`, compartilhados daemon↔CLI):

```
GET /api/config   → 200 ConfigView
PATCH /api/config → 200 ConfigView | 400 {error}
```

```go
type ConfigView struct {
    Judge  JudgeConfigView `json:"judge"`
    Hooks  []HookView      `json:"hooks"`
}
type JudgeConfigView struct {
    Language string `json:"language"`
    // F-006 adiciona: Prompt, PromptOverride
}
type HookView struct {
    Name    string         `json:"name"`
    Enabled bool           `json:"enabled"`
    Params  map[string]any `json:"params"` // valores atuais
    Schema  map[string]ParamSpec `json:"schema"` // para o CLI renderizar edição
}
type ParamSpec struct {
    Type    string  `json:"type"`    // "int" | "float" | "duration_s"
    Default float64 `json:"default"`
    Min     float64 `json:"min"`
}
// PATCH aceita parcial: {"judge":{"language":"es"}, "hooks":[{"name":"recall","enabled":false,"params":{"goldThreshold":1500}}]}
```

- **Hooks parametrizáveis** (interface opcional — não quebra os 8 implementors):

```go
// internal/hooks/config.go (novo)
type Configurable interface {
    Configure(params map[string]any) error // valida e aplica; erro → 400 no PATCH
    Spec() map[string]ParamSpec            // declara params suportados
}
```

Hooks que implementam `Configurable`: `Periodic5MinHook{IntervalSeconds}` (default 300, min 60 — `CurrentMark` passa a usar o campo), `RecallHook{GoldThreshold, MinGameTimeSeconds}`, `LaningPhaseEndHook{MarkSeconds}`. Demais hooks: apenas enable/disable.

- **Registry** ganha: `SetEnabled(name string, enabled bool) error`, `Configure(name string, params map[string]any) error`, `Snapshot() []HookView`. `Evaluate` pula hooks desabilitados. Registry passa a guardar entradas `{hook, enabled}` — mudança interna, assinatura de `Register`/`Evaluate`/`Hooks` preservada.

- **Reset de baseline:** ao habilitar um hook ou mudar seus params, `Orchestrator.ResetHook(name)` zera `prevFired[name]` para o mark atual (evita disparo retroativo). Novo método público no orchestrator.

- **RuntimeConfig** (`pkg/service/runtime_config.go`, novo): struct com `sync.RWMutex` segurando o último `ConfigView` aplicado; handlers HTTP leem dele, e o `apply` propaga para registry/builder/orchestrator. Fonte da verdade inicial = `DaemonConfig` (env/flags).

- **CLI** (tab Config): HTTP client `internal/tui/configclient.go` derivando endereço do WS (`ws://host:port/ws` → `http://host:port`). GET ao conectar; keys: `space` toggle hook, `e` edita param selecionado (textinput do bubbles, já no go.mod), `L` cicla idioma en→pt-BR→es (PATCH). Erros 400 exibidos na status bar.

## 4. Tasks

### T1 — Hooks configuráveis
- **Arquivos:** `internal/hooks/config.go` (novo), `periodic.go`, `event.go`
- Interface `Configurable`; campos + `Configure`/`Spec` nos 3 hooks listados; `Periodic5MinHook.CurrentMark` usa `IntervalSeconds`; validação de range em `Configure` (erro descritivo).
- **Testes:** table-driven por hook: params válidos aplicados; fora de range → erro; `ShouldFire` respeita novo intervalo/threshold.

### T2 — Registry: enable/disable + configure
- **Arquivo:** `internal/hooks/periodic.go` (Registry vive aqui)
- Entrada vira `{hook Hook, enabled bool}`; `SetEnabled`, `Configure` (type-assert `Configurable`, erro se não suportado), `Snapshot`; `Evaluate` filtra desabilitados.
- **Testes:** `registry_test.go` — hook desabilitado não dispara; `Configure` em hook não-configurável → erro.

### T3 — Orchestrator.ResetHook
- **Arquivo:** `internal/orchestrator/orchestrator.go`
- `ResetHook(name string)`: seta `prevFired[name] = CurrentMark` do hook no gameTime corrente (guarda `lastGameTime` no struct, atualizado a cada `Tick`).
- **Testes:** após reset, hook periódico não dispara retroativamente.

### T4 — RuntimeConfig + handlers HTTP
- **Arquivos:** `pkg/service/runtime_config.go` (novo), `pkg/service/config_api.go` (novo), `pkg/service/types.go` (DTOs)
- `RuntimeConfig` thread-safe; `handleGetConfig` e `handlePatchConfig` (decode JSON, aplica seções, responde `ConfigView` efetivo; validação falhou → 400 `{"error":...}`).
- **Testes:** `httptest`: GET inicial reflete env; PATCH parcial merge; PATCH inválido (hook inexistente, param fora de range, idioma inválido) → 400; concorrência GET/PATCH com `-race`.

### T5 — Wiring no daemon
- **Arquivo:** `pkg/service/daemon.go`
- `Daemon` guarda `builder` e `registry` como campos; `NewDaemon` cria `RuntimeConfig` a partir de `cfg`; `Run` registra rotas no mux; `applyConfig` chama `registry.SetEnabled/Configure`, `orch.ResetHook`, `builder.SetLanguage`.
- **Testes:** integração leve: daemon com config custom → GET reflete; PATCH desabilita hook → orchestrator não dispara mais.

### T6 — CLI: client HTTP + tab Config
- **Arquivos:** `internal/tui/configclient.go` (novo), `internal/tui/config.go` (novo, estado da tab), `internal/tui/model.go`, `internal/tui/messages.go`
- `TabConfig` (tabCount 5); mensagens `ConfigLoadedMsg`, `ConfigSavedMsg`, `ConfigErrorMsg`; render: tabela hook | enabled | params; footer com keys; edição via `textinput`.
- **Testes:** `model_test.go` — navegação para a tab; toggle emite PATCH (client mockado via interface `ConfigClient`); erro 400 aparece na status bar.

### T7 — Docs
- **Arquivos:** `docs/api-contract.md`, `README.md`
- Documentar `GET/PATCH /api/config` com exemplos de payload e tabela de params por hook.

## 5. Critérios de Aceite

1. `GET /api/config` retorna todos os 8 hooks com enabled/params/schema e o idioma atual.
2. `PATCH` desabilitando `player-death` → dicas de morte param imediatamente, sem restart.
3. `PATCH` com `intervalSeconds=120` no `periodic-5min` → próximo disparo ocorre na marca de 2min seguinte, sem disparo retroativo.
4. Idioma trocado via `PATCH` (ou tecla `L` no CLI) reflete na próxima dica.
5. Param inválido → 400 com mensagem clara; config anterior preservada.
6. Defaults na subida = comportamento atual (todos hooks habilitados, params atuais).
7. `go test -race ./...` limpo.

## 6. Dependências / Pontos de Contato

- **Consome F-003** (`SetLanguage`, `NormalizeLanguage`).
- **F-006 estende** `config_api.go` (seção `judge.prompt`) e a tab Config — arquivos criados aqui, editados lá. Executar antes.
- **F-005/F-007/F-008** tocam `daemon.go` em pontos distintos (tick/recorder); ordem sequencial evita conflito de merge.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.900 | Ciclos: 1*
