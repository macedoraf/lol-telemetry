# Feature 05 Specification: Live Client API Endpoint Abstractions

## 1. Objective
Abstract the individual endpoints of the Live Client Data API into the SDK so developers can fetch specific telemetry segments (like items, scores, or events) efficiently, rather than always fetching the full `allgamedata` payload.

## 2. Requirements

### 2.1 Endpoint Methods (`pkg/riotclient/client.go`)
Implement the following methods in the SDK client, returning their respective typed models from the existing `pkg/riotclient/models.go`:
- `GetActivePlayerName() (string, error)` — returns the raw string body from `/activeplayername`.
- `GetActivePlayerAbilities() (Abilities, error)` — `/activeplayerabilities`.
- `GetActivePlayerRunes() (FullRunes, error)` — `/activeplayerrunes`.
- `GetPlayerList() ([]AllPlayer, error)` — `/playerlist`.
- `GetPlayerScores(riotId string) (PlayerScores, error)` — `/playerscores?riotId={id}`.
- `GetPlayerSummonerSpells(riotId string) (SummonerSpells, error)` — `/playersummonerspells?riotId={id}`.
- `GetPlayerMainRunes(riotId string) (PlayerRunes, error)` — `/playermainrunes?riotId={id}`.
- `GetPlayerItems(riotId string) ([]Item, error)` — `/playeritems?riotId={id}`.
- `GetEventData() (Events, error)` — `/eventdata`.
- `GetGameStats() (GameData, error)` — `/gamestats`.

### 2.2 Internal Helper
- Add a private `getJSON(path, dest)` helper in `client.go` that:
  1. Builds the full URL from `BaseURL + path`.
  2. Executes the HTTP GET with the configured `HTTPClient`.
  3. Returns an error for non-2xx status codes.
  4. Decodes the JSON body into `dest`.
- The new endpoint methods should use this helper to avoid duplication.

### 2.3 Riot ID Parameter Handling
- For endpoints requiring a player identifier (Scores, Spells, Runes, Items), the parameter must be `riotId`.
- The client must correctly format the HTTP request (e.g., `?riotId=Riot%20Tuxedo%23TXC1`).
- Use `url.QueryEscape` or `url.Values` to encode the `riotId` parameter so spaces and `#` are handled correctly.

### 2.4 Model Adjustments
- `AllPlayer` already includes the Riot ID shim fields from Feature 04; no additional model changes are required.
- Reuse existing `models.go` types (`Abilities`, `FullRunes`, `PlayerScores`, `SummonerSpells`, `PlayerRunes`, `Item`, `Events`, `GameData`).

### 2.5 Mock Server Updates (`test/mocksrv/main.go`)
- Extend the mock API to serve sample responses for each new endpoint.
- Add static mock JSON files under `testdata/mocks/` for:
  - `activeplayername.json`
  - `activeplayerabilities.json`
  - `activeplayerrunes.json`
  - `playerlist.json`
  - `playerscores.json`
  - `playersummonerspells.json`
  - `playermainrunes.json`
  - `playeritems.json`
  - `eventdata.json`
  - `gamestats.json`
- Keep `MOCK_PAYLOAD` behavior for `/allgamedata` unchanged.

### 2.6 Testing
- Write unit tests for each new client method using `httptest` servers, validating:
  - Correct HTTP path (and query string for riotId methods).
  - Successful JSON decoding into the expected model.
  - Proper URL encoding of the `riotId` parameter.
- Add integration tests (build-tagged `integration`) that exercise the methods against the Docker Compose mock API.

## 3. Out of Scope
- Implementing WebSockets or external Riot API (cloud) abstractions. Only the local Live Client Data API is covered.
- Polling, caching, or TUI integration for these endpoints.

## 4. Acceptance Criteria
- [x] All 10 individual endpoints are implemented as distinct, typed methods in `pkg/riotclient/client.go`.
- [x] A private `getJSON` helper is used by each endpoint method.
- [x] Requests requiring a player ID correctly accept and URL-encode the `riotId`.
- [x] `test/mocksrv/main.go` handles requests to all individual routes.
- [x] Static mock JSON files exist for every endpoint.
- [x] Unit tests pass for all new methods.
- [x] `go test ./...` and the Docker Compose test stack pass.


## 5. Api Raw reference

````
GET ​https://127.0.0.1:2999/liveclientdata/activeplayername
Returns the player name.

"Riot Tuxedo#TXC1"
GET ​https://127.0.0.1:2999/liveclientdata/activeplayerabilities
Get Abilities for the active player.

{
    "E": {
        "abilityLevel": 0,
        "displayName": "Molten Shield",
        "id": "AnnieE",
        "rawDescription": "GeneratedTip_Spell_AnnieE_Description",
        "rawDisplayName": "GeneratedTip_Spell_AnnieE_DisplayName"
    },
    "Passive": {
        "displayName": "Pyromania",
        "id": "AnniePassive",
        "rawDescription": "GeneratedTip_Passive_AnniePassive_Description",
        "rawDisplayName": "GeneratedTip_Passive_AnniePassive_DisplayName"
    },
    "Q": {
        "abilityLevel": 0,
        "displayName": "Disintegrate",
        "id": "AnnieQ",
        "rawDescription": "GeneratedTip_Spell_AnnieQ_Description",
        "rawDisplayName": "GeneratedTip_Spell_AnnieQ_DisplayName"
    },
    "R": {
        "abilityLevel": 0,
        "displayName": "Summon: Tibbers",
        "id": "AnnieR",
        "rawDescription": "GeneratedTip_Spell_AnnieR_Description",
        "rawDisplayName": "GeneratedTip_Spell_AnnieR_DisplayName"
    },
    "W": {
        "abilityLevel": 0,
        "displayName": "Incinerate",
        "id": "AnnieW",
        "rawDescription": "GeneratedTip_Spell_AnnieW_Description",
        "rawDisplayName": "GeneratedTip_Spell_AnnieW_DisplayName"
    }
}
GET ​https://127.0.0.1:2999/liveclientdata/activeplayerrunes
Retrieve the full list of runes for the active player.

{
    "keystone": {
        "displayName": "Electrocute",
        "id": 8112,
        "rawDescription": "perk_tooltip_Electrocute",
        "rawDisplayName": "perk_displayname_Electrocute"
    },
    "primaryRuneTree": {
        "displayName": "Domination",
        "id": 8100,
        "rawDescription": "perkstyle_tooltip_7200",
        "rawDisplayName": "perkstyle_displayname_7200"
    },
    "secondaryRuneTree": {
        "displayName": "Sorcery",
        "id": 8200,
        "rawDescription": "perkstyle_tooltip_7202",
        "rawDisplayName": "perkstyle_displayname_7202"
    },
    "generalRunes": [
        {
            "displayName": "Electrocute",
            "id": 8112,
            "rawDescription": "perk_tooltip_Electrocute",
            "rawDisplayName": "perk_displayname_Electrocute"
        },
        ...
    ],
    "statRunes": [
        {
            "id": 5007,
            "rawDescription": "perk_tooltip_StatModCooldownReductionScaling"
        },
        {
            "id": 5008,
            "rawDescription": "perk_tooltip_StatModAdaptive"
        },
        {
            "id": 5003,
            "rawDescription": "perk_tooltip_StatModMagicResist"
        }
    ]
}
All Players
GET ​https://127.0.0.1:2999/liveclientdata/playerlist
Retrieve the list of heroes in the game and their stats.

[
    {
        "championName": "Annie",
        "isBot": false,
        "isDead": false,
        "items": [...],
        "level": 1,
        "position": "MIDDLE",
        "rawChampionName": "game_character_displayname_Annie",
        "respawnTimer": 0.0,
        "runes": {...},
        "scores": {...},
        "skinID": 0,
        "summonerName": "Riot Tuxedo",
        "riotId": "Riot Tuxedo#TXC1",
        "riotIdGameName": "Riot Tuxedo",
        "riotIdTagLine": "TXC1",
        "summonerSpells": {...},
        "team": "ORDER"
    },
    ...
]
GET ​https://127.0.0.1:2999/liveclientdata/playerscores?riotId=
Retrieve the list of the current scores for the player.

{
    "assists": 0,
    "creepScore": 0,
    "deaths": 0,
    "kills": 0,
    "wardScore": 0.0
}
GET ​https://127.0.0.1:2999/liveclientdata/playersummonerspells?riotId=
Retrieve the list of the summoner spells for the player.

{
    "summonerSpellOne": {
        "displayName": "Flash",
        "rawDescription": "GeneratedTip_SummonerSpell_SummonerFlash_Description",
        "rawDisplayName": "GeneratedTip_SummonerSpell_SummonerFlash_DisplayName"
    },
    "summonerSpellTwo": {
        "displayName": "Ignite",
        "rawDescription": "GeneratedTip_SummonerSpell_SummonerDot_Description",
        "rawDisplayName": "GeneratedTip_SummonerSpell_SummonerDot_DisplayName"
    }
}
GET ​https://127.0.0.1:2999/liveclientdata/playermainrunes?riotId=
Retrieve the basic runes of any player.

{
    "keystone": {
        "displayName": "Electrocute",
        "id": 8112,
        "rawDescription": "perk_tooltip_Electrocute",
        "rawDisplayName": "perk_displayname_Electrocute"
    },
    "primaryRuneTree": {
        "displayName": "Domination",
        "id": 8100,
        "rawDescription": "perkstyle_tooltip_7200",
        "rawDisplayName": "perkstyle_displayname_7200"
    },
    "secondaryRuneTree": {
        "displayName": "Sorcery",
        "id": 8200,
        "rawDescription": "perkstyle_tooltip_7202",
        "rawDisplayName": "perkstyle_displayname_7202"
    }
}
GET ​https://127.0.0.1:2999/liveclientdata/playeritems?riotId=
Retrieve the list of items for the player.

[
    {
        "canUse": true,
        "consumable": false,
        "count": 1,
        "displayName": "Warding Totem (Trinket)",
        "itemID": 3340,
        "price": 0,
        "rawDescription": "game_item_description_3340",
        "rawDisplayName": "game_item_displayname_3340",
        "slot": 6
    },
    ...
]
Events
GET ​https://127.0.0.1:2999/liveclientdata/eventdata
Get a list of events that have occurred in the game.

{
    "Events": [
        {
            "EventID": 0,
            "EventName": "GameStart",
            "EventTime": 0.0325561985373497
        },
        ...
    ]
}
You can find a list of sample events here.

Game
GET ​https://127.0.0.1:2999/liveclientdata/gamestats
Basic data about the game.

{
  "gameMode": "CLASSIC",
  "gameTime": 0.000000000,
  "mapName": "Map11",
  "mapNumber": 11,
  "mapTerrain": "Default"
}
Any of these endpoints that returned a summonerName, now return a RiotID shim over summonerName, and new fields called riotId, riotIdGameName and riotIdTagLine in structured responses. Any endpoints that took a SummonerName as a parameter now accepts only the riotId parameter. It attempts to match the name to RiotID first, then RiotIDGameName, then SummonerName (to maintain backwards compatibility until we can fully deprecate SummonerName).



```