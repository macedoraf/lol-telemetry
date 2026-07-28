# 008-timeseries-feature-engineering — Tasks de Implementação

## 1. Objetivo

Feature engineering sobre os dados brutos da Live Client Data API: o Judge passa a receber uma **série temporal de features derivadas** (gold/min, XP/min, spikes, objetivos/death timers via eventos, build/matchup), produzida por um pipeline **Transform** desacoplado do consumo. Schemas originais (`riotclient.AllGameData`, payload atual) **não são modificados** — tudo é aditivo.

## 2. Limitações da API (decisões de design honestas)

A Live Client Data API expõe `currentGold` **apenas para o jogador ativo**; demais jogadores têm apenas level, KDA, CS e itens. Consequências registradas:

| Capacidade pedida | Viável? | Estratégia |
|---|---|---|
| Gold/min do jogador | ✅ exato | `currentGold` direto |
| Gold/min do time | ❌ indisponível | não computado (documentado no prompt) |
| XP/min (jogador e todos) | ⚠️ estimado | curva estática nível→XP acumulado |
| Spikes de gold/XP individuais | ⚠️ proxy | completions de itens + level-ups por janela (mesma heurística dos hooks existentes) |
| Spikes de time / controle de objetivos | ✅ exato | eventos enriquecidos (F6): `DragonKill`/`BaronKill`/`HeraldKill`/`TurretKilled`/`InhibKilled`/`Ace` com `KillerName`/`Stolen`/`DragonType` |
| Death timers (10 jogadores) | ✅ exato | `isDead` + `respawnTimer` direto do payload (F6) |
| Build vs matchup vs sinergia | ✅ parcial | features estruturadas (itens, diffs, KDA/CS) + raciocínio do LLM |

## 3. Arquitetura (Transform Pattern)

```
[tick daemon] ──AllGameData──> [features.Tracker.Add]        (acúmulo, toda tick)
                                     │ ring buffer (janela)
[trigger fires] ──> [orchestrator] ──> [Pipeline.Compute]    (transformação, sob demanda)
                                     │ []Transformer
                                     v
                              types.FeatureVector ──> JudgeRequest.Features ──> LLM
                                     │
                                     └──> recorder (features.jsonl)
```

- **Separação consumo × transformação:** o orchestrator (consumo) não conhece fórmulas; cada `Transformer` é uma unidade pura `Window → features`. Adicionar capacidade nova = registrar um transformer novo, sem tocar em nada existente.
- **Sem ciclo de import:** `FeatureVector` mora em `internal/types` (que não importa `features`); `features` importa `types` + `riotclient`.

```go
// internal/features/transform.go
type Window interface {                      // visão read-only da série
    Samples() []Sample                       // ordenado por gameTime
    Last() (Sample, bool)
    Since(seconds float64) []Sample
    Events() []riotclient.Event              // última lista conhecida (deep copy; campos extras F6)
}
type Transformer interface {
    Name() string
    Transform(w Window, fv *types.FeatureVector) // preenche sua seção
}
type Pipeline struct{ t []Transformer }
func (p *Pipeline) Compute(w Window) types.FeatureVector
```

```go
// internal/features/tracker.go
type Sample struct {                         // extração mínima por tick (~barato)
    GameTime float64
    GameMode string                         // p/ correção de gold inicial em F1 (CLASSIC=500, ARAM=1400)
    Players  []PlayerSample                  // 10 jogadores
}
type PlayerSample struct {
    SummonerName, ChampionName, Team, Position string
    Level, Kills, Deaths, Assists, CS, ItemsCompleted int  // ItemsCompleted = riotclient.ItemCount(items): já exclui Consumable e ItemID==0
    ItemsGold    int                           // Σ price dos itens não-consumíveis (F3 por valor)
    IsActive     bool
    IsDead       bool
    RespawnTimer float64                       // >0 enquanto morto (death timers, F6)
    Gold         float64                       // >0 só p/ ativo
}
type Tracker struct {
    mu     sync.RWMutex                        // tick daemon (escrita) × triggers/leitores concorrentes
    events []riotclient.Event                  // última lista de eventos (cumulativa na API; substituída a cada Add)
    // ring buffer cap 3600 (1h @1s) + sessão
}
// Add usa Lock(); Window/Samples/Last/Since/Events usam RLock() e devolvem CÓPIAS (deep copy do slice).
func (t *Tracker) Add(data riotclient.AllGameData)
func (t *Tracker) Window() Window            // Since() aloca novo []Sample, nunca fatia o array subjacente
func (t *Tracker) Reset()                    // troca de sessão (zera ring + events)
```

```go
// internal/types/types.go — ADITIVO:
type FeatureVector struct {
    GameMinute  int                `json:"gameMinute"`
    WindowSec   float64            `json:"windowSec"`
    Samples     int                `json:"samples"`
    Player      PlayerFeatures     `json:"player"`
    Team        TeamFeatures       `json:"allyTeam"`
    Enemy       TeamFeatures       `json:"enemyTeam"`
    Matchup     *MatchupFeatures   `json:"matchup,omitempty"` // POINTER: nil ⇒ omitido (ARAM/AFK/rota indefinida)
}
type PlayerFeatures struct {
    GoldPerMin, XPPerMin            float64
    GoldDelta1m, GoldDelta5m        float64
    XPDelta1m, XPDelta5m            float64
    Level, ItemsCompleted           int
    ItemsGold                       int   // Σ price não-consumível (proxy de poder por valor)
    GoldSpike1m                     bool
}
type TeamFeatures struct {
    ItemCompletions1m int            `json:"itemCompletions1m"` // proxy de spike de gold individual
    LevelUps1m        int            `json:"levelUps1m"`        // proxy de spike de XP
    AvgXPPerMin       float64        `json:"avgXpPerMin"`
    Spikes            []string       `json:"spikes,omitempty"`  // ex: "Jinx completed item @14:02"
    Objectives        ObjectiveCount `json:"objectives"`        // F6: ground truth via eventos
    Kills1m           int            `json:"kills1m"`           // F6: kills do time na janela de 60s
    DeadNow           int            `json:"deadNow"`           // F6: mortos agora (isDead)
    MaxRespawnSec     float64        `json:"maxRespawnSec,omitempty"` // F6: maior respawnTimer vivo
}
type ObjectiveCount struct {          // F6: derivado dos eventos enriquecidos
    Dragons, Barons, Heralds, Towers, Inhibs int
    Steals    int  `json:"steals,omitempty"`    // Stolen=="True" atribuído ao time
    SoulPoint bool `json:"soulPoint,omitempty"` // Dragons==3 (SR; 4º = soul)
}
type MatchupFeatures struct {
    LevelDiff, CSDiff, ItemDiff int     `json:",omitempty"` // jogador − oponente de lane
    KillDiff                    int
    OpponentXPPerMin            float64
}
// JudgeRequest ganha: Features *FeatureVector `json:",omitempty"` (nil = desligado)
```

## 4. Catálogo de Features (entrada → transformação → justificativa)

> Versão normativa. Na implementação, copiar para `docs/features/008-timeseries-feature-engineering-catalog.md` com exemplos JSON reais extraídos de `testdata/mocks/`.

### F1 — Gold/min do jogador (cold-start corrigido por gameMode)
- **Entrada:** `activePlayer.currentGold=4200`, `gameData.gameTime=720`, `gameData.gameMode="CLASSIC"`.
- **Transformação:** `perMin(gold − StartingGold(mode), gameTime)`. `StartingGold`: `{"CLASSIC":500, "ARAM":1400}`, default `0` (modo desconhecido ⇒ sem correção, marcado no prompt). Ex.: `(4200−500)/12 = 308 g/min` (vs. `350` sem correção). Deltas `gold(t) − gold(t−60s)` e `−300s` via janela não sofrem cold start (são variações reais).
- **Safe-math:** `perMin` devolve `0` se `gameTime < 1.0` — evita `NaN`/`+Inf` que fazem `json.Marshal` falhar com `unsupported value`.
- **Justificativa:** ritmo de farm/renda é o indicador macro mais estável; descontar o gold inicial evita inflar a média nos primeiros minutos; deltas capturam aceleração recente (kills, objetivos) que a média esconde.

### F2 — XP/min (todos os jogadores; cold-start absorvido pela tabela)
- **Entrada:** `allPlayers[i].level=9`, `gameTime=720`.
- **Transformação:** `xpEst = XP_TABLE[level]` com `XP_TABLE[1] = 0` por construção (jogador nasce no nível 1 com 0 XP acumulado) ⇒ `perMin(xpEst, gameTime)` já ignora o "XP inicial" sem subtração extra. Tabela estática em `xp.go` (curva oficial de Summoner's Rift; `// ponytail: aproximação — XP real não é exposto; erro típico <5% pós-lane`). Deltas 1m/5m idem F1. Safe-math: `perMin` retorna `0` se `gameTime < 1.0`.
- **Justificativa:** vantagem de nível precede power spikes; XP/min do oponente alimenta o matchup. Definir `XP_TABLE[1]=0` substitui a subtração `−XP_LEVEL_1` sugerida inicialmente (matematicamente equivalente e mais simples).

### F3 — Spike de gold/XP (aliado e inimigo; consumíveis já excluídos)
- **Entrada:** série de `ItemsCompleted` e `Level` por jogador na janela de 60s. `PlayerSample.ItemsCompleted` é populado por `riotclient.ItemCount(items)`, que já descarta `Consumable==true` (wards 3340, poções 2003, biscoitos 2010) e `ItemID==0`. A Live Client Data API expõe `consumable` e `price` por item (ver `testdata/mocks/allgamedata.json`): filtro local, **sem chamada externa**.
- **Transformação:** `itemCompletions1m = Σ max(0, ItemsCompleted(t) − ItemsCompleted(t−60))` por time; `levelUps1m` idem para `Level`; `Spikes` lista strings com champion+timestamp. Jogador ativo também tem `GoldSpike1m = goldDelta1m > 1200` (const `GoldSpikeThreshold`, documentada). `PlayerSample.ItemsGold` (Σ `price` não-consumível) é extraído por tick e exposto em `PlayerFeatures.ItemsGold` como proxy de poder por valor — spike por gold de item vira `ItemsGold(t) − ItemsGold(t−60)` quando necessário, sem mudança de schema. Spikes **de time** (objetivos) são ground truth via F6, não proxy.
- **Justificativa:** completion de item é o evento discreto mais próximo de "spike de gold" disponível na API; alinha-se à heurística já validada nos hooks `ally-gold-spike`/`enemy-gold-spike`. Reusar `ItemCount` elimina a história de "poção como power spike" sem inventar filtro novo.

### F4 — Matchup de lane (omitido quando o oponente não existe)
- **Entrada:** jogador ativo vs `FindOpponent` (mesma posição, time oposto) na última amostra. `FindOpponent` passa a retornar `(AllPlayer, bool)`; `false` cobre ARAM (Position=""), early-game sem lane inferida, AFKs e modos sem lane definida.
- **Transformação:** se encontrado, diffs escalares (level, CS, itens, kills) + `OpponentXPPerMin` (F2). Se NÃO encontrado, o transformer atribui `fv.Matchup = nil` e o campo é omitido do JSON (`Matchup *MatchupFeatures` + `omitempty`). O LLM lida melhor com ausência explícita do que com diffs zeros contra um "fantasma".
- **Migração:** o único caller existente de `FindOpponent` é `EnemyGoldSpikeHook` (`internal/hooks/event.go:157`); trocar `opponent := riotclient.FindOpponent(...)` + checagem `SummonerName==""` por `(opponent, ok)`, com `ok=false` ⇒ hook não dispara (comportamento idêntico ao atual).
- **Justificativa:** o LLM raciocina melhor sobre diffs do que sobre valores absolutos cruzados; omitir matchup em ARAM evita alucinação de comparação contra ninguém.

### F5 — Build vs composições (sinergia)
- **Entrada:** listas de itens (nome/ID) dos 10 jogadores + `itemsCompleted` por time.
- **Transformação:** sem fórmula — features estruturadas (itens por jogador já existem no payload; F5 agrega `ItemsCompleted` por time e o `ItemDiff` do matchup). O raciocínio matchup×sinergia fica com o LLM sobre esse material.
- **Justificativa:** tags de itens (AP/AD/tank) não existem na API; derivá-las exigiria base estática de itens — fora de escopo (candidata a F-009).

### F6 — Objetivos, kills e death timers (ground truth via eventos enriquecidos)
- **Entrada:** `events.Events[]` — a API envia campos extras por tipo (ver sample oficial `liveclientdata_sample.json` em developer.riotgames.com/docs/lol), hoje **descartados** por `riotclient.Event`: `DragonKill{DragonType, Stolen, KillerName, Assisters}`, `BaronKill`/`HeraldKill{Stolen, KillerName, Assisters}`, `TurretKilled{TurretKilled, KillerName, Assisters}`, `InhibKilled{InhibKilled, KillerName}`, `ChampionKill{VictimName, KillerName, Assisters}`, `Multikill{KillStreak}`, `Ace{AcingTeam}`, `FirstBrick{KillerName}`. Mais `isDead`/`respawnTimer` dos 10 jogadores na última amostra.
- **Atribuição de time (heurística documentada):** dragon/baron/herald → time do `KillerName` (lookup em `AllPlayers`; não encontrado ⇒ não conta, ex.: executado por minion); torre/inibidor → o identificador codifica o **dono** (`Turret_T1_*`/`Barracks_T1*` = ORDER, `_T2` = CHAOS) ⇒ crédito ao time **oposto**; kill → time do `KillerName`; ace → `AcingTeam` direto; `Stolen=="True"` incrementa `Steals` do time creditado.
- **Transformação:** agregação pura sobre `Window.Events()` (sem janela — a lista é cumulativa por sessão) + janela de 60s para `Kills1m` (`EventTime ≥ t−60`); `DeadNow`/`MaxRespawnSec` do `Last()`. `SoulPoint = Dragons==3`. Spike strings para steals/aces (ex.: "Baron stolen by CHAOS @28:41").
- **Justificativa:** eventos são a **única fonte de ground truth** de objetivos na API — substituem qualquer proxy para spikes de time (dragon/baron = gold global real) e alimentam as dicas macro mais acionáveis ("2 inimigos mortos, um por 30s ⇒ forçar baron"). Custo zero de polling: já vêm no `/allgamedata`. Corrige de brinde o legado `JudgeRequest.Objectives` (hoje hardcoded zerado em `builder.go`) — ver T3.
- **Compliance:** todos esses dados já são visíveis in-game (kill feed + scoreboard); a política de Game Integrity só proíbe o que a API nem expõe (ex.: tracking de ult/cooldown inimigo).

## 5. Tasks

### T1 — Tracker + janela (thread-safe + deep copy)
- **Arquivos:** `internal/features/tracker.go`, `tracker_test.go`
- Ring buffer 3600; `Add` extrai `Sample` do `AllGameData` (incluindo `GameMode` em `Sample` p/ F1, `IsDead`/`RespawnTimer`/`ItemsGold` p/ F6) sem alocar histórico de structs grandes; armazena a **última** `[]riotclient.Event` (substituição, não append — a lista é cumulativa na API); `Reset` na troca de sessão (zera ring + events); `Window` com `Since` + `Events`.
- **Concorrência:** `sync.RWMutex` no `Tracker`; `Add` usa `Lock()`; `Window()`/`Samples()`/`Last()`/`Since()` usam `RLock()`. `Since()`/`Samples()` **devolvem deep copy do slice** (alocam `make([]Sample, n)` e copiam, nunca fatiam o array subjacente do ring) — senão o daemon sobrescreve o conteúdo enquanto o Transformer lê, gerando data race intermitente. Hoje Add e Compute rodam na mesma goroutine (poll loop); o mutex blinda leitores concorrentes futuros (ex.: handler `/features` de debug).
- **Testes:** table-driven: extração de amostra (inclui `ItemsGold`, `IsDead`, `RespawnTimer`); wrap do ring; `Since` com bordas; janela com 0/1 amostras; `Events()` devolve deep copy e reflete o último `Add`; `go test -race` rodando `Add` numa goroutine e `Window`/`Since`/`Events` em outra (trava o contrato mesmo hoje).

### T2 — Tabela XP + transformadores F1–F4, F6 (safe-math + ARAM + matchup omitido)
- **Arquivos:** `internal/features/xp.go`, `gold.go`, `spikes.go`, `matchup.go`, `objectives.go` (F6), `transform.go`, `pkg/riotclient/models.go` (`FindOpponent` + extensão aditiva de `Event`), `internal/hooks/event.go` (migração do caller), `testdata/mocks/allgamedata.json` (eventos enriquecidos) + testes
- Helper `perMin(value, gameSeconds float64) float64`: `if gameSeconds < 1.0 { return 0 }; return value / (gameSeconds/60)` — usado por F1 e F2; previne `NaN`/`+Inf` (`json.Marshal` falha com `unsupported value`).
- `StartingGold(mode string) float64`: `{"CLASSIC":500, "ARAM":1400}`, default `0` (modo desconhecido ⇒ sem correção, marcado no prompt do LLM). F1: `perMin(gold − StartingGold(mode), gameTime)`.
- `XP_TABLE[1] = 0` por construção (cold-start de XP absorvido pela tabela; substitui a subtração `−XP_LEVEL_1`). F2: `perMin(xpEst, gameTime)`.
- F3: `PlayerSample.ItemsCompleted = riotclient.ItemCount(items)` (reusar helper existente que já filtra `Consumable`).
- F4: `Matchup *MatchupFeatures` (pointer + `omitempty`); `FindOpponent` retorna `(AllPlayer, bool)`; nil quando `Position=""` (ARAM/AFK). Migrar o caller em `event.go` (`EnemyGoldSpikeHook`).
- F6: `riotclient.Event` ganha campos aditivos (todos `omitempty`): `KillerName`, `VictimName`, `Assisters`, `DragonType`, `Stolen` (string `"True"`/`"False"` na API), `TurretKilled`, `InhibKilled`, `AcingTeam`, `KillStreak`. Mock `allgamedata.json` ganha `DragonKill` (com `Stolen`) e `TurretKilled` baseados no sample oficial — preserva a regra do api-contract (DTO ⇔ mock).
- F6 transformer (`objectives.go`): agrega `ObjectiveCount` por time (regras de atribuição do catálogo), `Kills1m` por janela, `DeadNow`/`MaxRespawnSec` do `Last()`; Spike strings para steals/aces.
- `Pipeline` ordenado; cada transformer puro e independente; constantes exportadas e documentadas (`GoldSpikeThreshold=1200`, janelas 60/300s).
- **Testes:** table-driven por transformer com séries sintéticas: gold/min com `StartingGold(CLASSIC)=500` e `(ARAM)=1400`; `perMin` retorna 0 p/ `gameTime < 1.0`; XP/min com `XP_TABLE[1]=0` (nível 1 ⇒ 0 XP/min, sem div by zero); spike só quando threshold cruza; diffs com sinal correto; **matchup nil quando Position=""**; série curta (2 amostras) não panics; F6: atribuição por `KillerName`, torre `_T1_` ⇒ crédito CHAOS, `Stolen=="True"` ⇒ `Steals`, `Kills1m` respeita borda da janela, `DeadNow`/`MaxRespawnSec` do último sample, killer desconhecido ⇒ não conta.

### T3 — Integração orchestrator + types
- **Arquivos:** `internal/types/types.go` (FeatureVector + campo `Features` + `Matchup *MatchupFeatures`), `internal/orchestrator/orchestrator.go`
- `NewOrchestrator` aceita `pipeline *features.Pipeline` + `tracker` (nil ⇒ Features nil, comportamento atual intacto). No disparo de trigger: `req.Features = &fv` antes de `judge.Evaluate`. Builder **não muda**: `Judge.buildPrompt` faz `json.MarshalIndent(req, ...)`, então `Features` e `Matchup` entram no prompt automaticamente quando preenchidos e são omitidos pelo `omitempty` quando nil. Quando pipeline ativo, o orchestrator também preenche o legado `req.Objectives` (hoje hardcoded zerado em `builder.go`) mapeando `fv.Team.Objectives`/`fv.Enemy.Objectives` → ORDER/CHAOS conforme o time do jogador — elimina o campo sempre-zero **somente no modo ligado** (desligado segue byte-a-byte idêntico).
- **Testes:** pipeline nil ⇒ request byte-a-byte idêntico ao atual (regressão); pipeline + matchup encontrado ⇒ `Features` e `Matchup` presentes no prompt (fake LLM devolve o payload recebido); pipeline + matchup ausente (Position="") ⇒ `Matchup` ausente do JSON do prompt; pipeline ativo ⇒ `req.Objectives` preenchido coerente com o time do jogador.

### T4 — Wiring no daemon + env
- **Arquivos:** `pkg/service/config.go` (`LOL_FEATURES_ENABLED`, default `false`), `pkg/service/daemon.go`
- Quando ligado: cria tracker+pipeline; todo tick com sucesso ⇒ `tracker.Add` (mesma goroutine do `orch.Tick`/`Compute` hoje — mutex de T1 blinda futuros leitores concorrentes); troca/fim de sessão ⇒ `tracker.Reset` (reutiliza o `SessionManager` da F-005 **se** recording estiver on; caso contrário, detector local mínimo: `gameTime` retrocede ⇒ Reset).
- **Testes:** integração com mock HTTP: features chegam ao Judge fake; flag off ⇒ fluxo atual preservado.

### T5 — Gravação de features.jsonl
- **Arquivos:** `internal/recorder/features.go` (novo), `pkg/service/daemon.go`
- `RecordFeatures(fv)` no writer existente, **somente** quando `LOL_RECORD_ENABLED` (F-005) estiver on; grava a cada disparo de trigger **e** a cada marca cheia de 60s. Linha com `v,type,session,gameTime` (mesma chave de correlação da F-007) + `features`.
- **Testes:** linha válida correlacionável com `telemetry.jsonl` e `tips.jsonl` da mesma sessão.

### T6 — Catálogo normativo + docs
- **Arquivos:** `docs/features/008-timeseries-feature-engineering-catalog.md` (novo), `README.md`, `docs/api-contract.md`, `docs/features/tracking.md`
- Catálogo da seção 4 com exemplos JSON reais de `testdata/mocks/`: incluir F1 cold-start (`gameTime=0.5` ⇒ `goldPerMin=0` por safe-math), F4 com matchup e F4 sem matchup (ARAM), F6 com os eventos enriquecidos do mock (DragonKill roubado + TurretKilled ⇒ contagens por time) e death timers. Documentar `LOL_FEATURES_ENABLED`, a tabela `StartingGold` por `gameMode`, `XP_TABLE` com fonte + `XP_TABLE[1]=0`, os campos extras de `Event` (fonte: sample oficial) e as regras de atribuição de time.

## 6. Critérios de Aceite

1. `LOL_FEATURES_ENABLED=false` (default): `JudgeRequest` byte-a-byte equivalente ao atual — zero regressão.
2. Ligado: toda dica do Judge contém `features` com gold/min, XP/min, diffs de matchup (quando há oponente), spikes coerentes com a série, objetivos por time e death timers (validado em teste com mock).
3. Nenhum schema original (`riotclient`, payload atual) modificado — apenas campos aditivos com `omitempty` (e `FindOpponent` migrada para `(AllPlayer, bool)` sem mudar o comportamento do hook existente).
4. Novo transformer se registra com 1 linha no `Pipeline` (demonstrar no teste do T3).
5. `features.jsonl` correlacionável via `(session, gameTime)` com telemetria e dicas.
6. Custo por tick desprezível: 1 extração de `Sample` (10 jogadores, campos escalares); transformação só ocorre em triggers e na marca de 60s.
7. `go test -race ./...` limpo **com `LOL_FEATURES_ENABLED=true`** (cobre Add×Window concorrente e o mutex de T1).
8. Safe-math: para qualquer `gameTime >= 0` (inclusive `0.0`), nenhum campo de `FeatureVector` é `NaN`/`+Inf` — marshalling JSON sempre succeeds (`perMin` em F1/F2 + `StartingGold`).
9. `Matchup` ausente (ARAM / Position indefinida) ⇒ campo **omitido** do JSON do prompt, nunca uma struct zeros ("fantasma").
10. F-008 não cria novos endpoints externos: apenas o Live Client Data API (Port 2999). Metadata de itens (tags, build paths) fica como follow-up (Data Dragon, §8).
11. F6 ground truth: com o mock enriquecido (`DragonKill` `Stolen="True"` por CHAOS + `TurretKilled` de `Turret_T1_*`), o vetor expõe `enemyTeam.objectives.dragons=1`, `enemyTeam.objectives.steals=1`, `enemyTeam.objectives.towers=1`; o legado `req.Objectives` reflete os mesmos números no modo ligado.
12. Death timers: com um jogador `isDead=true, respawnTimer=28`, o time correspondente expõe `deadNow≥1` e `maxRespawnSec≥28`.

## 7. Dependências / Pontos de Contato

- **Depende de F-005** (sessão + recorder) e **F-007** (chave de correlação). Executar por último, como no roadmap.
- Edita `orchestrator.go`, `pkg/riotclient/models.go` (`FindOpponent`) e `internal/hooks/event.go` (caller) depois de F-004/F-006/F-007 — todos os pontos de edição são aditivos/migratórios e listados acima para evitar conflito de merge.
- Fora de escopo (registrar como follow-up): gold/min de time exato (impossível na API atual — spikes de time cobertos por F6 via eventos); base estática de itens e campeões para sinergia real (F-009 → §8).

## 8. Gap de Itens — follow-up de pesquisa (postergado para F-009)

A Live Client Data API entrega `consumable` e `price` por item no payload (ver `testdata/mocks/allgamedata.json`), o que basta para F-008: filtro de consumíveis (já em `riotclient.ItemCount`) e detecção de spike por contagem. O que **não** vem da Live Client e motivou este follow-up:

- **Tags de item** (AP/AD/tank, mythic/legendary, build paths, passivas): ausentes na Live Client Data API. Necessárias para análise real de sinergia (F5 do catálogo).
- **Fonte recomendada:** **Data Dragon** — CDN público e **sem autenticação** (referenciado pela Riot em `https://developer.riotgames.com/docs`), serve JSONs estáticos de itens e campeões versionados por patch. Cabe num client HTTP simples com cache local por patch — **não** exige Riot API key nem integração LCU/OAuth.
- **Riot APIs autenticadas (Match v5, etc.):** descartadas para este fim — exigem `X-Riot-Token`, têm rate limit e são pós-partida, não tempo real. Justificam-se só para análise histórica (post-game), não para o Judge em tempo real.
- **Ação concreta em F-009:** `pkg/items/static.go` carregando `item.json` do Data Dragon (cache em `~/.lol-telemetry/cache/items-<patch>.json`); estende `ItemSnapshot` com `Tags []string` e `Tier string` (tudo `omitempty`). Idem `champion.json` (tags Fighter/Mage/… por campeão) para análise de composição. Sem cache populado, as features degradam graciosamente para a contagem atual — F-008 não fica refém de F-009.

---

*Refinado por OpenCode Agent (revisão F-008 v3: +F6 eventos enriquecidos — objetivos/kills/death timers, ItemsGold, Objectives legado; v2: safe-math, mutex, ARAM, matchup omitido, gap de itens) | Tokens estimados: ~4.600 | Ciclos: 3*
