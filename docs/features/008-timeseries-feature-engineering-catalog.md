# 008-timeseries-feature-engineering — Catálogo de Features

Este documento é a versão normativa do catálogo de features derivadas da Live Client Data API. Os exemplos JSON são ilustrativos; para exemplos reais do mock, veja `testdata/mocks/allgamedata.json` (baseline) e `testdata/mocks/allgamedata-enriched.json` (F6).

## F1 — Gold/min do jogador (cold-start corrigido por gameMode)

**Entrada:** `activePlayer.currentGold=4200`, `gameData.gameTime=720`, `gameData.gameMode="CLASSIC"`.

**Transformação:** `perMin(gold − StartingGold(mode), gameTime)`.

`StartingGold`: `{"CLASSIC":500, "ARAM":1400}`, default `0`.

Exemplo: `(4200−500)/12 = 308 g/min` (vs. `350` sem correção).

**Safe-math:** `perMin` devolve `0` se `gameTime < 1.0`.

```json
{
  "player": {
    "goldPerMin": 308.33,
    "goldDelta1m": 900,
    "goldDelta5m": 1600,
    "goldSpike1m": false
  }
}
```

## F2 — XP/min (todos os jogadores; curva estática por nível)

**Entrada:** `allPlayers[i].level=9`, `gameTime=720`.

**Transformação:** `XP_TABLE[level]`; `XP_TABLE[1]=0` por construção.

```json
{
  "player": {
    "xpPerMin": 420.00,
    "xpDelta1m": 120,
    "xpDelta5m": 480
  },
  "allyTeam": {
    "avgXpPerMin": 395.5
  },
  "enemyTeam": {
    "avgXpPerMin": 410.2
  }
}
```

## F3 — Spikes de gold/XP individuais

**Entrada:** série de `ItemsCompleted` e `Level` por jogador na janela de 60s.

`ItemsCompleted` é `riotclient.ItemCount(items)` (descarta consumíveis e `ItemID==0`). `ItemsGold` é `Σ price` dos itens não-consumíveis.

```json
{
  "player": {
    "itemsCompleted": 2,
    "itemsGold": 1500
  },
  "allyTeam": {
    "itemCompletions1m": 1,
    "levelUps1m": 0,
    "spikes": ["Jinx completed item @14:02"]
  }
}
```

## F4 — Matchup de lane (omitido quando não existe)

**Entrada:** jogador ativo vs oponente de mesma posição/time oposto.

```json
{
  "matchup": {
    "levelDiff": -1,
    "csDiff": -10,
    "itemDiff": 1,
    "killDiff": 1,
    "opponentXpPerMin": 510.0
  }
}
```

Em ARAM/posição vazia/AFK, `matchup` é omitido do JSON:

```json
{
  "gameMinute": 10,
  "windowSec": 60,
  "samples": 60,
  "player": {...},
  "allyTeam": {...},
  "enemyTeam": {...}
}
```

## F5 — Build vs composições

**Entrada:** listas de itens (nome/ID) dos 10 jogadores + `itemsCompleted`/`ItemsGold` por time.

**Transformação:** sem fórmula — features estruturadas + raciocínio do LLM. Tags de itens (AP/AD/tank) e campeões são follow-up de F-009 via Data Dragon.

## F6 — Objetivos, kills e death timers (ground truth via eventos)

**Entrada:** `events.Events[]` com campos enriquecidos:

```json
[
  {"EventID":1,"EventName":"DragonKill","EventTime":312.0,"DragonType":"Infernal","Stolen":"True","KillerName":"Enemy Ahri","Assisters":["Enemy Ahri"]},
  {"EventID":2,"EventName":"TurretKilled","EventTime":315.0,"TurretKilled":"Turret_T1_C_05_A","KillerName":"Enemy Ahri","Assisters":["Enemy Ahri"]},
  {"EventID":3,"EventName":"ChampionKill","EventTime":580.0,"VictimName":"Riot Tuxedo","KillerName":"Enemy Ahri","Assisters":["Enemy Ahri"]},
  {"EventID":4,"EventName":"Ace","EventTime":590.0,"AcingTeam":"CHAOS"}
]
```

**Atribuição de time:**
- Dragon/Baron/Herald: time do `KillerName` (lookup em `AllPlayers`).
- Turret/Inhibitor: estrutura `T1`/`Barracks_T1` pertence ao ORDER ⇒ crédito ao CHAOS; `T2` ⇒ crédito ao ORDER.
- Ace: `AcingTeam` direto.

```json
{
  "enemyTeam": {
    "objectives": {
      "dragons": 1,
      "barons": 0,
      "heralds": 0,
      "towers": 1,
      "inhibs": 0,
      "steals": 1,
      "soulPoint": false
    },
    "kills1m": 1,
    "deadNow": 1,
    "maxRespawnSec": 28.5,
    "spikes": [
      "Infernal dragon stolen by CHAOS @05:12",
      "Turret killed by CHAOS @05:15",
      "Ace by CHAOS @09:50"
    ]
  }
}
```

## Configuração

| Variável | Default | Descrição |
|---|---|---|
| `LOL_FEATURES_ENABLED` | `false` | Liga/desliga o pipeline de features. |
| `LOL_RECORD_ENABLED` | `false` | Necessário para gravar `features.jsonl`. |
| `LOL_RECORDINGS_DIR` | `./recordings` | Diretório base das sessões. |

## Arquivos de gravação

`<sessionID>/telemetry.jsonl`, `<sessionID>/tips.jsonl`, `<sessionID>/features.jsonl` são correlacionáveis por `(session, gameTime)`.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.800 | Ciclos: 1*
