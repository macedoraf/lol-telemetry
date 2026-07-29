# 005-telemetry-recording — Tasks de Implementação

## 1. Objetivo

Persistir os dados brutos da Live Client Data API para uso futuro (inclui F-007 e F-008). Opt-in via env, formato **JSONL** append-only, escrita **100% fora da thread principal** (canal bufferizado + goroutine dedicada; drop com contagem em caso de backpressure — nunca bloqueia o poll).

## 2. Estado Atual

- `pkg/service/daemon.go` — `tick()` chama `client.GetGameData()` a cada `PollInterval` (default 1s) e descarta o raw após montar o `GameState`.
- `pkg/service/config.go` — sem variáveis de gravação.
- Não existe conceito de sessão de jogo. F-007 (correlação) e F-008 (série temporal) dependem dele — esta feature o cria.

## 3. Design

- **Formato JSONL**: uma linha JSON por snapshot; append barato, streamable, legível com `jq`. Vetores `.json` (array único) exigiriam reescrita — descartado.

```json
{"v":1,"type":"telemetry","ts":1750000000000,"session":"20260727-143012-a1b2c3","gameTime":612.4,"data":{ ...allgamedata bruto... }}
```

- **Layout em disco:** `<dir>/<sessionID>/telemetry.jsonl` (F-007 grava `tips.jsonl` e F-008 `features.jsonl` no mesmo diretório da sessão).
- **Sessão** (`internal/recorder/session.go`): transição inativo→ativo (`gameTime` sai de `<=0`/erro para `>0`) **ou** `gameTime` retrocede (partida nova) ⇒ nova sessão. ID = `YYYYMMDD-HHMMSS-<rand6 hex>` (crypto/rand). Fim de sessão: `gameTime <= 0` ou erro de API ⇒ flush + fecha arquivo atual.
- **Recorder** (`internal/recorder/recorder.go`):

```go
type Recorder struct { /* chan, goroutine, bufio.Writer, métricas */ }
func New(dir string) (*Recorder, error)              // cria dir; goroutine de escrita
func (r *Recorder) StartSession(id string) error     // abre <dir>/<id>/telemetry.jsonl
func (r *Recorder) EndSession()                      // flush + close
func (r *Recorder) Record(rec TelemetryRecord)       // non-blocking; drop + DropCount++
func (r *Recorder) Close(ctx context.Context) error  // drain + flush final
```

  - Canal com cap **1024** (~17min de buffer a 1s). Cheio ⇒ drop, incrementa contador, loga 1x a cada 100 drops. `// ponytail: drop policy; se perda for inaceitável, trocar por spillover em disco`.
  - Escrita: `bufio.Writer` (32KB), flush a cada **5s** por ticker na goroutine, no fim de sessão e no `Close`.
- **SessionManager** é consumido pelo daemon mesmo quando recording está off? **Não** — lazy: sessão só existe dentro do recorder. F-008 (tracker) terá seu próprio detector ou reutilizará via injeção (decisão registrada na F-008).
- **Env** (`pkg/service/config.go`): `LOL_RECORD_ENABLED` (default `false`), `LOL_RECORDINGS_DIR` (default `./recordings`).

## 4. Tasks

### T1 — SessionManager
- **Arquivo:** `internal/recorder/session.go` (novo)
- `type SessionManager struct { active bool; sessionID string; lastGameTime float64 }`; `Observe(gameTime float64, apiOK bool) (id string, started bool, ended bool)` implementando as transições da seção 3; gerador de ID com crypto/rand.
- **Testes:** table-driven: início de partida; fim (gameTime→0 e erro de API); nova partida (gameTime retrocede); mesma partida não duplica sessão.

### T2 — Recorder
- **Arquivo:** `internal/recorder/recorder.go` (novo), `internal/recorder/types.go` (novo: `TelemetryRecord` com tags JSON)
- Implementação conforme seção 3; métricas internas `Written`, `Dropped` acessíveis para teste/log.
- **Testes** (herméticos, `t.TempDir()`): grava N linhas e re-lê validando JSON por linha e campos obrigatórios; canal cheio ⇒ `Dropped>0` e `Record` retorna imediatamente (teste com timeout de 100ms); `Close` drena tudo; `-race` limpo.

### T3 — Config
- **Arquivo:** `pkg/service/config.go`
- `DaemonConfig` ganha `RecordEnabled bool`, `RecordingsDir string`; parsing das envs.
- **Testes:** table-driven em `config_test.go`.

### T4 — Wiring no daemon
- **Arquivo:** `pkg/service/daemon.go`
- `NewDaemon`: se `RecordEnabled`, cria `Recorder` (falha ⇒ log + segue sem gravar, nunca derruba o daemon). `tick()`: após `GetGameData` com sucesso → `Observe` + `StartSession`/`EndSession` conforme transições + `Record` com o `AllGameData` bruto; erro de API ⇒ `Observe(0,false)`. `Run`: `defer d.recorder.Close(ctx)` ao sair. Log de sessão: `recording session <id> started/ended (written=N dropped=M)`.
- **Testes:** integração com mock HTTP do riotclient: daemon grava arquivo na sessão; com `RecordEnabled=false` nenhum arquivo é criado.

### T5 — Docs
- **Arquivos:** `README.md`, `docs/api-contract.md`
- Envs novas; formato da linha JSONL; layout de diretório; exemplo `jq` de leitura.

## 5. Critérios de Aceite

1. Com `LOL_RECORD_ENABLED=false` (default): zero arquivos criados, zero overhead relevante (1 `if` por tick).
2. Com recording on: `telemetry.jsonl` contém 1 linha válida por poll, com `data` fiel ao JSON bruto da API.
3. Poll nunca bloqueia: canal saturado ⇒ drops contados e logados, tick continua.
4. Trocar de partida ⇒ novo diretório de sessão; fim de partida ⇒ arquivo fechado e flushed.
5. Shutdown (SIGINT/SIGTERM) ⇒ nenhuma linha bufferizada é perdida.
6. `go test -race ./internal/recorder/...` limpo.

## 6. Dependências / Pontos de Contato

- **F-007** reusará `Recorder` para `tips.jsonl` (mesmo diretório de sessão) — expõe `SessionID() string` no Recorder (adicionar em T2, trivial) para correlação.
- **F-008** reusará o writer para `features.jsonl` e o `sessionID`.
- Toca `daemon.go` e `config.go` — mesmos arquivos de F-004/F-006; ordem sequencial definida no roadmap (004 → 005 → 006) evita conflito.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.600 | Ciclos: 1*
