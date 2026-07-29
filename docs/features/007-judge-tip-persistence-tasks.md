# 007-judge-tip-persistence — Tasks de Implementação

## 1. Objetivo

Gravar cada dica do Judge em arquivo, de forma **correlacionável** com a telemetria bruta da F-005: mesma sessão + `gameTime` como chave de join.

## 2. Estado Atual

- `internal/orchestrator/orchestrator.go` — `Result{HookName, GameMinute, Advice, Reasoning}`; **sem** `gameTime` e sem a pergunta que originou a dica.
- `pkg/service/daemon.go` — após `orch.Tick`, dicas vão só para o WebSocket (`BroadcastAdvice`).
- `internal/recorder` (F-005) — `Recorder` com writer JSONL por sessão e `SessionID()`.

## 3. Design

- **Arquivo:** `<dir>/<sessionID>/tips.jsonl` (mesmo diretório da telemetria da sessão — vizinhança física = primeira forma de correlação).

```json
{"v":1,"type":"tip","ts":1750000001234,"session":"20260727-143012-a1b2c3","gameTime":612.4,"gameMinute":10,"hookName":"periodic-5min","question":"Evaluate the current macro state...","advice":"...","reasoning":"..."}
```

- **Chave de correlação:** `(session, gameTime)`. Telemetria é gravada a cada poll (~1s); a dica nasce de um tick específico, então o join é *nearest-neighbor* em `telemetry.jsonl` com tolerância = `PollInterval`. Exemplo (vai na doc):

```sh
jq -c 'select(.session=="S" and (.gameTime - T | . * . < 1))' recordings/S/telemetry.jsonl
```

- **Sem sessão ativa, sem gravação:** se o recorder estiver entre sessões (bug ou timing), a dica é descartada com log warn — dica sem telemetria não tem valor analítico.
- **Gating:** segue `LOL_RECORD_ENABLED` (mesmo recorder, mesmo canal — dicas entram na fila existente, não criam goroutine nova). Mesma política de drop da F-005.
- **`orchestrator.Result`** ganha `GameTime float64` e `Question string` (aditivo; consumidor atual — daemon — ignora campos extras sem quebrar).

## 4. Tasks

### T1 — Enriquecer orchestrator.Result
- **Arquivo:** `internal/orchestrator/orchestrator.go`
- `Result` += `GameTime float64` (do `data.GameData.GameTime` do tick) e `Question string` (`trigger.Question`).
- **Testes:** `orchestrator_test.go` — result contém gameTime do tick e a pergunta do hook.

### T2 — Recorder: writer de dicas
- **Arquivos:** `internal/recorder/tips.go` (novo), `internal/recorder/recorder.go`
- `func (r *Recorder) RecordTip(t TipRecord)` — mesma fila/canal da telemetria (campo `type` discrimina na hora de escrever); `StartSession` passa a abrir também `tips.jsonl` lazy na primeira dica (evita arquivo vazio em toda sessão). `TipRecord` com tags JSON da seção 3.
- **Testes** (`t.TempDir()`): dica gravada cria `tips.jsonl` com linha válida contendo `session` e `gameTime`; sessão sem dicas ⇒ sem `tips.jsonl`; `Close` não perde dicas bufferizadas.

### T3 — Wiring no daemon
- **Arquivo:** `pkg/service/daemon.go`
- No loop de `responses` do `tick`: após `BroadcastAdvice`, se recorder ativo e sessão aberta ⇒ `RecordTip` com todos os campos; sessão fechada ⇒ warn.
- **Testes:** integração com mock HTTP: hook dispara ⇒ `tips.jsonl` contém a dica com `session` igual ao do `telemetry.jsonl` da mesma partida e `gameTime` dentro de 1×PollInterval de alguma linha de telemetria (validação do join).

### T4 — Docs
- **Arquivos:** `docs/api-contract.md`, `README.md`
- Schema de `tips.jsonl`; chave de correlação; exemplo `jq` de join dica↔telemetria.

## 5. Critérios de Aceite

1. Toda dica broadcastada durante uma sessão gravada aparece em `tips.jsonl`.
2. `session` + `gameTime` permitem localizar a linha de telemetria correspondente (tolerância = PollInterval).
3. Com `LOL_RECORD_ENABLED=false`: nenhum arquivo, nenhum overhead além de um `if`.
4. Dica fora de sessão ⇒ log warn, nenhum crash, daemon segue.
5. Nenhuma goroutine ou arquivo novo além dos já criados pela F-005.

## 6. Dependências / Pontos de Contato

- **Depende de F-005** (Recorder, sessão, layout de diretório).
- **F-008** gravará `features.jsonl` no mesmo diretório e incluirá a mesma chave `(session, gameTime)` — padrão definido aqui é pré-condição para a correlação dica↔features↔telemetria.
- Edita `orchestrator.go` e `daemon.go` antes da F-008 (que também os toca) — ordem sequencial do roadmap.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.300 | Ciclos: 1*
