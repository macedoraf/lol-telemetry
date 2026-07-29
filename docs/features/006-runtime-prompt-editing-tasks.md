# 006-runtime-prompt-editing — Tasks de Implementação

## 1. Objetivo

Alterar o prompt de sistema do Judge em runtime, sem redeploy: daemon aceita `PATCH /api/config` na seção `judge.prompt` (reuso do endpoint criado na F-004); CLI de testes exibe o prompt efetivo e edita via `$EDITOR`.

## 2. Estado Atual

- `internal/judge/payload/builder.go` — `defaultSystemPrompt()` (após F-003: `systemPrompt(lang)`) é a única fonte de prompt; recompilação + restart para mudar.
- `pkg/service/config_api.go` + `runtime_config.go` — criados na F-004; seção `judge` já existe com `language`.
- `internal/tui/config.go` — tab Config criada na F-004.

## 3. Design

- **Override, não replace de código:** `promptOverride` vazio ⇒ comportamento atual (prompt default + diretiva de idioma). Não vazio ⇒ override vira o **prompt base**; a diretiva de idioma da F-003 é **sempre anexada ao final** (preserva F-003 em qualquer cenário).
- **Validação no PATCH:** trim não-vazio, ≤ 4000 chars. Falha ⇒ `400 {"error":...}`, config anterior preservada. Reset para o default: `PATCH {"judge":{"prompt":""}}` (string vazia = limpa override — único caso em que vazio é aceito).
- **Concorrência:** `Builder` ganha `promptOverride atomic.Value` (mesmo padrão do `lang` da F-003): leitura na goroutine de poll, escrita na goroutine HTTP.
- **GET expõe os dois:** `promptOverride` (o que o usuário setou) e `effectivePrompt` (o que vai ao LLM, já com diretiva de idioma) — facilita debug no CLI.

```go
// pkg/service/types.go — JudgeConfigView estendido:
type JudgeConfigView struct {
    Language        string `json:"language"`
    PromptOverride  string `json:"promptOverride"`
    EffectivePrompt string `json:"effectivePrompt"`
}

// internal/judge/payload/builder.go:
func (b *Builder) SetPromptOverride(prompt string) error // valida; "" limpa
func (b *Builder) EffectivePrompt() string
```

## 4. Tasks

### T1 — Builder: prompt override
- **Arquivo:** `internal/judge/payload/builder.go`
- Campo `promptOverride atomic.Value`; `SetPromptOverride` (validação: trim, 4000 max, retorna erro descritivo); `systemPrompt()` usa override como base quando presente e sempre anexa diretiva de idioma; `EffectivePrompt()` expõe o resultado.
- **Testes:** table-driven em `builder_test.go`: sem override → default+idioma; com override → override+idioma; override + troca de idioma → diretiva atualiza; `SetPromptOverride("  ")→erro`; `>4000 chars→erro`; `""`→limpa.

### T2 — API: seção judge.prompt
- **Arquivos:** `pkg/service/runtime_config.go`, `pkg/service/config_api.go`, `pkg/service/types.go`
- `RuntimeConfig` guarda o override; `JudgeConfigView` estendido; PATCH valida via `builder.SetPromptOverride` e faz rollback no `RuntimeConfig` se o builder rejeitar.
- **Testes (httptest):** PATCH com prompt válido → GET reflete override e effective; inválido → 400 + config anterior intacta; `prompt:""` → volta ao default; PATCH de idioma + prompt na mesma chamada → ambos aplicados.

### T3 — CLI: visualizar e editar prompt
- **Arquivos:** `internal/tui/config.go`, `internal/tui/messages.go`
- Seção "Judge" na tab Config: idioma atual + prompt efetivo (scroll com viewport do bubbles se exceder a área). Tecla `p`: escreve o prompt efetivo em arquivo temp, abre `$EDITOR` (suspende o programa Bubble Tea com `tea.ExecProcess`), ao sair lê o arquivo; não-vazio e diferente ⇒ PATCH; `$EDITOR` ausente ⇒ hint na status bar ("set $EDITOR or use curl"). Arquivo temp sempre removido.
- **Testes:** fluxo de edição com editor runner injetável (`var editorRunner = func(path string) error` — fake nos testes); cancelamento (arquivo inalterado) ⇒ nenhum PATCH.

### T4 — Docs
- **Arquivos:** `docs/api-contract.md`, `README.md`
- Exemplo `curl` de PATCH do prompt; regras de validação; precedência override×idioma; como resetar.

## 5. Critérios de Aceite

1. Prompt alterado via PATCH vale na **próxima** dica, sem restart/redeploy.
2. Diretiva de idioma (F-003) permanece anexada mesmo com override ativo.
3. Prompt inválido ⇒ 400 e o prompt anterior continua em uso.
4. `prompt:""` restaura o default empacotado.
5. GET mostra `promptOverride` e `effectivePrompt` coerentes com o que o Judge recebe (comparável via log debug).
6. `go test -race ./...` limpo.

## 6. Dependências / Pontos de Contato

- **Depende de F-004** (endpoint, RuntimeConfig, tab Config) e **coopera com F-003** (diretiva de idioma).
- Edita `builder.go` (após F-003) e `config_api.go`/`runtime_config.go` (após F-004) — mudanças aditivas, sem reescrita dos trechos anteriores.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.300 | Ciclos: 1*
