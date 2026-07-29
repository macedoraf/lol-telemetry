# 003-judge-tip-language — Tasks de Implementação

## 1. Objetivo

Selecionar o idioma das dicas do Judge. Idiomas suportados no lançamento: `en` (default), `pt-BR`, `es`. Seleção via env `JUDGE_LANGUAGE` no daemon; override em runtime chega na F-004 (endpoint de config), para o qual esta feature deixa o seam pronto (`Builder.SetLanguage`).

## 2. Estado Atual (código relevante)

- `pkg/service/config.go` — `LoadDaemonConfigFromEnv()` monta `DaemonConfig`. **Sem** campo de idioma.
- `pkg/service/daemon.go` — `NewDaemon` cria `payload.NewBuilder()` (linha ~89) sem parâmetros.
- `internal/judge/payload/builder.go` — `defaultSystemPrompt()` retorna prompt fixo em inglês; `Build()` injeta `SystemPrompt` no `types.JudgeRequest`.
- `internal/judge/judge.go` — envia `req.SystemPrompt` ao LLM. **Não muda.**

## 3. Design

- O idioma é resolvido **uma única vez** no prompt de sistema (payload builder). Judge, orchestrator e LLM client permanecem agnósticos.
- Idioma inválido → fallback `en` + log warn.
- `Builder` armazena o idioma em `atomic.Value`: o builder é lido pela goroutine de poll e será escrito pela goroutine HTTP na F-004. Sem mutex dedicado.
- As chaves do JSON de resposta (`advice`, `reasoning`) permanecem em inglês — apenas o **conteúdo** muda de idioma.

## 4. Interfaces

```go
// internal/judge/payload/builder.go
func NewBuilder(language string) *Builder
func (b *Builder) SetLanguage(lang string)   // thread-safe (atomic.Value); valida; fallback "en"
func (b *Builder) Language() string

// pkg/service/config.go — DaemonConfig ganha:
JudgeLanguage string // env JUDGE_LANGUAGE; default "en"
```

Mapeamento idioma → diretiva no prompt (const privado em builder.go):

```go
var languageNames = map[string]string{
    "en":    "English",
    "pt-BR": "Brazilian Portuguese",
    "es":    "Spanish",
}
```

`systemPrompt()` = `defaultSystemPrompt()` + `"\nRespond entirely in <LanguageName>. JSON keys must remain in English."`

Validação centralizada: `func NormalizeLanguage(lang string) (string, bool)` no pacote `payload` (reutilizada pelo PATCH da F-004).

## 5. Tasks

### T1 — Config: env `JUDGE_LANGUAGE`
- **Arquivo:** `pkg/service/config.go`
- Ler `JUDGE_LANGUAGE`; validar contra `en|pt-BR|es` (case-sensitive `pt-BR`); inválido → `"en"`. Popular `DaemonConfig.JudgeLanguage`.
- **Testes:** `pkg/service/config_test.go` (novo, table-driven): unset → `en`; `pt-BR` → `pt-BR`; `fr` → `en`; vazio → `en`.

### T2 — Builder com idioma
- **Arquivo:** `internal/judge/payload/builder.go`
- `NewBuilder(language string)`; campo `lang atomic.Value` (armazena string já normalizada); `SetLanguage`/`Language`; `defaultSystemPrompt(lang)` passa a montar prompt com a diretiva de idioma.
- **Testes:** `builder_test.go` — table-driven: para cada idioma, `Build()` produz `SystemPrompt` contendo o nome do idioma e a frase `JSON keys must remain in English`; idioma inválido no construtor → prompt em inglês.

### T3 — Wiring no daemon
- **Arquivo:** `pkg/service/daemon.go`
- `NewDaemon`: `payload.NewBuilder(cfg.JudgeLanguage)`; log de boot inclui `lang=%s`.
- **Testes:** coberto por T1/T2; sem teste novo (wiring).

### T4 — Doc de variáveis de ambiente
- **Arquivo:** `README.md` (seção de configuração existente)
- Documentar `JUDGE_LANGUAGE` com valores aceitos e default.

## 6. Critérios de Aceite

1. Sem `JUDGE_LANGUAGE`, dicas saem em inglês (comportamento atual preservado).
2. `JUDGE_LANGUAGE=pt-BR` → `SystemPrompt` instrui resposta em português; `es` em espanhol.
3. Valor inválido → inglês + warn no log do daemon.
4. `SetLanguage` é seguro para chamada concorrente com `Build` (go test -race limpo).
5. Nenhuma mudança em `internal/judge`, `internal/orchestrator`, `internal/hooks`, `pkg/riotclient`.

## 7. Dependências / Pontos de Contato

- **F-004** chamará `Builder.SetLanguage` a partir do `PATCH /api/config` (o daemon precisará guardar referência ao builder — registrado na spec F-004, T2).
- Conflito de arquivo com **F-006** em `builder.go`: F-006 adiciona `promptOverride`; executar na ordem definida (003 → 004 → 006). Os campos são independentes.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.400 | Ciclos: 1*
