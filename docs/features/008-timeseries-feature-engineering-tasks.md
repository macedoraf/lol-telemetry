# 008-timeseries-feature-engineering — Tasks de Implementação

## 1. Objetivo

Feature engineering sobre os dados brutos da Live Client Data API: o Judge passa a receber uma **série temporal de features derivadas** (gold/min, XP/min, spikes, build/matchup), produzida por um pipeline **Transform** desacoplado do consumo. Schemas originais (`riotclient.AllGameData`, payload atual) **não são modificados** — tudo é aditivo.

## 2. Limitações da API (decisões de design honestas)

A Live Client Data API expõe `currentGold` **apenas para o jogador ativo**; demais jogadores têm apenas level, KDA, CS e itens. Consequências registradas:

| Capacidade pedida | Viável? | Estratégia |
|---|---|---|
| Gold/min do jogador | ✅ exato | `currentGold` direto |
| Gold/min do time | ❌ indisponível | não computado (documentado no prompt) |
| XP/min (jogador e todos) | ⚠️ estimado | curva estática nível→XP acumulado |
| Spikes de gold/XP aliado e inimigo | ⚠️ proxy | completions de itens + level-ups por janela (mesma heurística dos hooks existentes) |
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
    Players  []PlayerSample                  // 10 jogadores
}
type PlayerSample struct {
    SummonerName, ChampionName, Team, Position string
    Level, Kills, Deaths, Assists, CS, ItemsCompleted int
    IsActive bool
    Gold     float64                         // >0 só p/ ativo
}
type Tracker struct { /* ring buffer cap 3600 (1h @1s) + sessão */ }
func (t *Tracker) Add(data riotclient.AllGameData)
func (t *Tracker) Window() Window
func (t *Tracker) Reset()                    // troca de sessão
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
    Matchup     MatchupFeatures    `json:"matchup"`
}
type PlayerFeatures struct {
    GoldPerMin, XPPerMin            float64
    GoldDelta1m, GoldDelta5m        float64
    XPDelta1m, XPDelta5m            float64
    Level, ItemsCompleted           int
    GoldSpike1m                     bool
}
type TeamFeatures struct {
    ItemCompletions1m int      `json:"itemCompletions1m"` // proxy de spike de gold
    LevelUps1m        int      `json:"levelUps1m"`        // proxy de spike de XP
    AvgXPPerMin       float64  `json:"avgXpPerMin"`
    Spikes            []string `json:"spikes,omitempty"`  // ex: "Jinx completed item @14:02"
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

### F1 — Gold/min do jogador
- **Entrada:** `activePlayer.currentGold=4200`, `gameData.gameTime=720`.
- **Transformação:** `gold / (gameTime/60)` → `4200/12 = 350 g/min`. Deltas: `gold(t) − gold(t−60s)` e `−300s` via janela.
- **Justificativa:** ritmo de farm/renda é o indicador macro mais estável; deltas capturam aceleração recente (kills, objetivos) que a média esconde.

### F2 — XP/min (todos os jogadores)
- **Entrada:** `allPlayers[i].level=9`, `gameTime=720`.
- **Transformação:** `xpEst = XP_TABLE[level]` (tabela estática nível→XP acumulado em `xp.go`, curva oficial de Summoner's Rift; `// ponytail: aproximação — XP real não é exposto; erro típico <5% pós-lane`) → `xpEst / (gameTime/60)`. Deltas 1m/5m idem F1.
- **Justificativa:** vantagem de nível precede power spikes; XP/min do oponente alimenta o matchup.

### F3 — Spike de gold/XP (aliado e inimigo)
- **Entrada:** série de `itemsCompleted` e `level` por jogador na janela de 60s.
- **Transformação:** `itemCompletions1m = Σ max(0, items(t)−items(t−60))` por time; `levelUps1m` idem para level; `Spikes` lista strings com champion+timestamp. Jogador ativo também tem `GoldSpike1m = goldDelta1m > 1200` (const `GoldSpikeThreshold`, documentada).
- **Justificativa:** completion de item é o evento discreto mais próximo de "spike de gold" disponível na API; alinha-se à heurística já validada nos hooks `ally-gold-spike`/`enemy-gold-spike`.

### F4 — Matchup de lane
- **Entrada:** jogador ativo vs `FindOpponent` (mesma posição, time oposto) na última amostra.
- **Transformação:** diffs escalares (level, CS, itens, kills) + `OpponentXPPerMin` (F2).
- **Justificativa:** o LLM raciocina melhor sobre diffs do que sobre valores absolutos cruzados.

### F5 — Build vs composições (sinergia)
- **Entrada:** listas de itens (nome/ID) dos 10 jogadores + `itemsCompleted` por time.
- **Transformação:** sem fórmula — features estruturadas (itens por jogador já existem no payload; F5 agrega `ItemsCompleted` por time e o `ItemDiff` do matchup). O raciocínio matchup×sinergia fica com o LLM sobre esse material.
- **Justificativa:** tags de itens (AP/AD/tank) não existem na API; derivá-las exigiria base estática de itens — fora de escopo (candidata a F-009).

## 5. Tasks

### T1 — Tracker + janela
- **Arquivos:** `internal/features/tracker.go`, `tracker_test.go`
- Ring buffer 3600; `Add` extrai `Sample` do `AllGameData` (sem alocar histórico de structs grandes); `Reset` na troca de sessão; `Window` com `Since`.
- **Testes:** table-driven: extração de amostra; wrap do ring; `Since` com bordas; janela com 0/1 amostras.

### T2 — Tabela XP + transformadores F1–F4
- **Arquivos:** `internal/features/xp.go`, `gold.go`, `spikes.go`, `matchup.go`, `transform.go` + testes
- `Pipeline` ordenado; cada transformer puro e independente; constantes exportadas e documentadas (`GoldSpikeThreshold=1200`, janelas 60/300s).
- **Testes:** table-driven por transformer com séries sintéticas: gold/min exato; XP/min pela tabela; spike só quando threshold cruza; diffs de matchup com sinal correto; série curta (2 amostras) não panics.

### T3 — Integração orchestrator + types
- **Arquivos:** `internal/types/types.go` (FeatureVector + campo `Features`), `internal/orchestrator/orchestrator.go`
- `NewOrchestrator` aceita `pipeline *features.Pipeline` + `tracker` (nil ⇒ Features nil, comportamento atual intacto). No disparo de trigger: `req.Features = &fv` antes de `judge.Evaluate`. Builder **não muda** (o JSON do request já é marshaled pelo Judge — `omitempty` esconde quando nil).
- **Testes:** com pipeline nil ⇒ request idêntico ao atual (regressão); com pipeline ⇒ `Features` populado e presente no prompt (via fake LLM).

### T4 — Wiring no daemon + env
- **Arquivos:** `pkg/service/config.go` (`LOL_FEATURES_ENABLED`, default `false`), `pkg/service/daemon.go`
- Quando ligado: cria tracker+pipeline; todo tick com sucesso ⇒ `tracker.Add`; troca/fim de sessão ⇒ `tracker.Reset` (reutiliza o `SessionManager` da F-005 **se** recording estiver on; caso contrário, detector local mínimo: `gameTime` retrocede ⇒ Reset).
- **Testes:** integração com mock HTTP: features chegam ao Judge fake; flag off ⇒ fluxo atual preservado.

### T5 — Gravação de features.jsonl
- **Arquivos:** `internal/recorder/features.go` (novo), `pkg/service/daemon.go`
- `RecordFeatures(fv)` no writer existente, **somente** quando `LOL_RECORD_ENABLED` (F-005) estiver on; grava a cada disparo de trigger **e** a cada marca cheia de 60s. Linha com `v,type,session,gameTime` (mesma chave de correlação da F-007) + `features`.
- **Testes:** linha válida correlacionável com `telemetry.jsonl` e `tips.jsonl` da mesma sessão.

### T6 — Catálogo normativo + docs
- **Arquivos:** `docs/features/008-timeseries-feature-engineering-catalog.md` (novo), `README.md`, `docs/api-contract.md`
- Catálogo da seção 4 com exemplos JSON reais de `testdata/mocks/` (entrada bruta, saída transformada, justificativa, limitações). Documentar `LOL_FEATURES_ENABLED` e a tabela XP com fonte.

## 6. Critérios de Aceite

1. `LOL_FEATURES_ENABLED=false` (default): `JudgeRequest` byte-a-byte equivalente ao atual — zero regressão.
2. Ligado: toda dica do Judge contém `features` com gold/min, XP/min, diffs de matchup e spikes coerentes com a série (validado em teste com mock).
3. Nenhum schema original (`riotclient`, payload atual) modificado — apenas campos aditivos com `omitempty`.
4. Novo transformer se registra com 1 linha no `Pipeline` (demonstrar no teste do T3).
5. `features.jsonl` correlacionável via `(session, gameTime)` com telemetria e dicas.
6. Custo por tick desprezível: 1 extração de `Sample` (10 jogadores, campos escalares); transformação só ocorre em triggers e na marca de 60s.
7. `go test -race ./...` limpo.

## 7. Dependências / Pontos de Contato

- **Depende de F-005** (sessão + recorder) e **F-007** (chave de correlação). Executar por último, como no roadmap.
- Edita `orchestrator.go` e `daemon.go` depois de F-004/F-006/F-007 — todos os pontos de edição são aditivos e estão listados acima para evitar conflito de merge.
- Fora de escopo (registrar como follow-up): base estática de itens para sinergia real; gold/min de time (impossível na API atual).

---

*Gerado por OpenCode Agent | Tokens estimados: ~2.300 | Ciclos: 1*
