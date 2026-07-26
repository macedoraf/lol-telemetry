# API Contract & Mocks (`api-contract.md`)

## 1. Overview
The `lol-telemetry` daemon communicates exclusively with the local **Live Client Data API** provided by the League of Legends game client. It exposes the processed data to external tools via a local WebSocket.

*   **Live Client Data API Base URL:** `https://127.0.0.1:2999/liveclientdata`
*   **Protocol:** HTTPS (Self-signed certificate, requires skipping SSL validation in the HTTP client).
*   **Authentication:** None (Local API).

*   **Daemon WebSocket:** `ws://localhost:8080/ws`
*   **Daemon Health:** `http://localhost:8080/health`

## 2. Core Endpoint Used in MVP
*   **Endpoint:** `/allgamedata`
*   **Full URL:** `https://127.0.0.1:2999/liveclientdata/allgamedata`
*   **Description:** Returns all available game data in a single JSON payload, including active player metadata, full player list, game time, and event list.

## 3. WebSocket Protocol

All messages are JSON objects with the following envelope:

```json
{
  "type": "game_state",
  "payload": { ... },
  "seq": 42,
  "ts": 1710000000000
}
```

### Message types

| Type | Description | Frequency |
|------|-------------|-----------|
| `hello` | Sent on every new connection | Once per connection |
| `game_state` | Current game snapshot | Every poll interval (default 1s) |
| `judge_advice` | Advice from the Judge | When a hook fires (e.g., every 5 min) |
| `event` | Recent game event | When a new event appears since the last tick |
| `error` | Service-level error | On Live Client Data API failures |

### `hello` payload

```json
{
  "version": "1.0.0",
  "serverTs": 1710000000000,
  "protocol": "lol-telemetry/ws/v1"
}
```

### `game_state` payload

```json
{
  "gameTime": 312.7,
  "gameMode": "CLASSIC",
  "mapName": "Map11",
  "players": [
    {
      "summonerName": "Player1",
      "championName": "Ashe",
      "team": "ORDER",
      "position": "BOTTOM",
      "level": 9,
      "cs": 82,
      "kills": 1,
      "deaths": 0,
      "assists": 3,
      "currentGold": 3200,
      "items": [ { "id": 1055, "name": "Doran's Blade", "slot": 0, "canUse": true, "cooldown": 0 } ],
      "runes": { "keystone": "Press the Attack", "primary": ["Precision"], "secondary": ["Inspiration"] },
      "isActive": true
    }
  ],
  "events": [ { "eventID": 1, "eventName": "GameStart", "eventTime": 0.0 } ],
  "timestamp": 1710000000000
}
```

### `judge_advice` payload

```json
{
  "hookName": "periodic-5min",
  "gameMinute": 5,
  "advice": "Recall before dragon; enemy bot has no sums.",
  "timestamp": 1710000000000
}
```

### `event` payload

```json
{
  "eventID": 12,
  "eventName": "ChampionKill",
  "eventTime": 312.7,
  "timestamp": 1710000000000
}
```

### `error` payload

```json
{
  "code": "lcu_error",
  "message": "Get \"https://127.0.0.1:2999/liveclientdata/allgamedata\": connection refused"
}
```

## 4. Mock Strategy & Test Data
To enable offline development, unit testing, and TDD without requiring an active League of Legends match, raw JSON responses must be stored locally.

*   **Mock File Path:** `testdata/mocks/allgamedata.json`
*   **Rule for AI/Developers:** Any DTO (Data Transfer Object) in `pkg/riotclient` MUST map strictly to the structure present in `testdata/mocks/allgamedata.json`. Do not infer fields outside of this contract.

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.200 | Ciclos: 1*