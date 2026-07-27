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

## 5. Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `LOL_DAEMON_PORT` | `8080` | WebSocket server port |
| `LOL_POLL_INTERVAL` | `1s` | Live Client Data API polling interval |
| `LOL_BASE_URL` | `https://127.0.0.1:2999/liveclientdata` | LoL API base URL |
| `OPENROUTER_API_KEY` | — | API key for Judge |
| `OPENROUTER_MODEL` | `openai/gpt-4o-mini` | Judge model |
| `JUDGE_ENABLED` | `true` if key is set | Toggle Judge |
| `LOL_CLI_LOG` | — | Override CLI log path |
| `LOL_DAEMON_LOG` | — | Override daemon log path |
| `LOL_RECORD_ENABLED` | `false` | Persist raw telemetry snapshots to disk |
| `LOL_RECORDINGS_DIR` | `./recordings` | Base directory for recorded sessions |
| `EDITOR` | — | External editor for the TUI prompt editor |

### PATCH judge prompt

```bash
curl -X PATCH http://localhost:8080/api/config \
  -H 'Content-Type: application/json' \
  -d '{"judge":{"prompt":"Focus only on dragon control."}}'
```

* Validation: non-empty and ≤ 4000 characters after trimming. Empty string resets to default.
* `GET /api/config` returns both `promptOverride` (user value) and `effectivePrompt` (value sent to the LLM, including the language directive).

### Recording format

When `LOL_RECORD_ENABLED=true`, each game session is written to `<LOL_RECORDINGS_DIR>/<sessionID>/telemetry.jsonl` as one JSON object per line:

```json
{"v":1,"type":"telemetry","ts":1750000000000,"session":"20260727-143012-a1b2c3","gameTime":612.4,"data":{...allgamedata bruto...}}
```

* `sessionID` = `YYYYMMDD-HHMMSS-hex12`, generated when the daemon first sees `gameTime > 0`.
* A new session is also created when `gameTime` goes backwards, indicating a new match.
* If the write channel backs up, records are dropped and counted; the poll loop never blocks.

### Judge tips

When `LOL_RECORD_ENABLED=true` and a Judge hook fires, a line is appended to `<sessionID>/tips.jsonl`:

```json
{"v":1,"type":"tip","ts":1750000001234,"session":"20260727-143012-a1b2c3","gameTime":612.4,"gameMinute":10,"hookName":"periodic-5min","question":"Evaluate the current macro state...","advice":"...","reasoning":"..."}
```

Correlation with telemetry is by `(session, gameTime)`:

```bash
SESSION=20260727-143012-a1b2c3
TIP_TIME=612.4
jq -c "select(.session==\"$SESSION\" and (.gameTime - $TIP_TIME | . * . < 1))" recordings/$SESSION/telemetry.jsonl
```

## 6. Troubleshooting & Logs

Both `lol-cli` and `lol-daemon` write startup logs to a file:

| Binary | Default log path (Windows) | Default log path (macOS/Linux) | Override |
|--------|---------------------------|-------------------------------|----------|
| `lol-cli` | `<exe-dir>\lol-cli.log` (falls back to temp) | `/tmp/lol-cli.log` | `LOL_CLI_LOG` |
| `lol-daemon` | `<exe-dir>\lol-daemon.log` (falls back to temp) | `/tmp/lol-daemon.log` | `LOL_DAEMON_LOG` |

### Test the LoL connection

```bash
# Test the default LoL API endpoint
lol-daemon -check

# Test a custom endpoint
lol-daemon -check -lol-url https://localhost:2999/liveclientdata
```

The daemon will print `LoL API connection OK` if the API is reachable and a game is active, or the exact error otherwise.

### If the TUI opens and closes immediately on Windows

Check the log file for the exact error. Common causes:

- The daemon is not running on `ws://localhost:8080/ws`.
- The terminal does not support the alternate screen buffer; try `lol-cli --no-alt`.
- The TUI is being launched without a real TTY; in that case `lol-cli` automatically falls back to `--raw` mode.

### If the connection with LoL stopped working

- Make sure a League of Legends match is in progress. The Live Client Data API is only active during games.
- Run `lol-daemon -check` to see the exact error.
- Check `lol-daemon.log` for repeated `LoL API connection failed` messages.
- Try changing the base URL with `LOL_BASE_URL` or `-lol-url` if your client binds to a different address.

### `connectex: Nenhuma conexão pôde ser feita` / `connectex: No connection could be made`

This is the Windows error for "connection refused". It means nothing is listening on the configured address. Common fixes:

1. **Start a League of Legends match.** The Live Client Data API only exists while you are in-game.
2. **Check the port.** The default is `2999`. If your LoL client is configured differently, set the correct URL:
   ```powershell
   $env:LOL_BASE_URL="https://localhost:2999/liveclientdata"
   ./lol-daemon.exe
   ```
3. **Firewall/antivirus.** Make sure Windows Defender or third-party antivirus is not blocking local HTTPS traffic on port 2999.
4. **Try localhost.** Some Windows installs resolve `127.0.0.1` differently. The daemon now automatically falls back to `localhost` if `127.0.0.1` fails at startup.
5. **Run with debug logs** to see the exact request:
   ```powershell
   ./lol-daemon.exe -check -debug
   ```

---

*Gerado por OpenCode Agent | Tokens estimados: ~1.800 | Ciclos: 4*