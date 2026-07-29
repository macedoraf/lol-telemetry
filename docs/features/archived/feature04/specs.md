# Feature 04 Specification: Core Data Model Adherence (`allgamedata`)

## 1. Objective
Ensure the SDK's core data models strictly adhere to the `allgamedata` JSON contract provided by the League of Legends Live Client Data API, guaranteeing accurate telemetry parsing without data loss.

## 2. Requirements

### 2.1 Data Structures (`pkg/riotclient/models.go`)
- Create or update Go structs to map the exact payload of the `allgamedata` endpoint.
- **`ActivePlayer`**: Must map `abilities`, `championStats`, `currentGold`, `fullRunes`, `level`, and `summonerName`.
- **`AllPlayers`**: Must map arrays of players including `championName`, `items`, `runes`, `scores`, `summonerSpells`, `team`, and the new Riot ID fields (`riotId`, `riotIdGameName`, `riotIdTagLine`).
- **`Events`**: Must map the `Events` array (e.g., `EventID`, `EventName`, `EventTime`).
- **`GameData`**: Must map `gameMode`, `gameTime`, `mapName`, `mapNumber`, `mapTerrain`.
- Define reusable nested structs: `Ability`, `ChampionStats`, `FullRunes`, `Rune`, `RuneTree`, `StatRune`, `Item`, `PlayerScores`, `SummonerSpells`, `SummonerSpell`, and `Event`.
- Keep all nested structs in `models.go` so `client.go` remains focused on HTTP logic.

### 2.2 JSON Unmarshaling & Validation
- Ensure all struct fields have correct `json:"fieldName"` tags.
- Use appropriate Go data types (e.g., `float64` for stats like `armor` and `gameTime`, `int` for `level` and `EventID`, `bool` for `isDead`).
- Use pointers or `omitempty` where appropriate for optional fields (e.g., Riot IDs) so missing fields do not cause unmarshaling errors.

### 2.3 Mock Data
- Enrich `testdata/mocks/allgamedata.json` to include at least one player with the new Riot ID fields (`riotId`, `riotIdGameName`, `riotIdTagLine`) so the new struct fields are covered by the primary test fixture.
- Preserve the existing Annie payload shape to avoid breaking fixtures used by other features.

### 2.4 Testing
- Create a test file (`pkg/riotclient/allgamedata_test.go`) that loads and unmarshals `testdata/mocks/allgamedata.json` into the new models.
- Write unit tests that assert nested fields (e.g., `activePlayer.championStats.abilityPower`, `allPlayers[0].runes.keystone.id`, `allPlayers[0].scores.kills`).
- Add a dedicated test case for missing optional fields (e.g., a player payload without `riotId*`) to prove unmarshaling succeeds and the fields are empty/zero.
- Keep tests in-memory and free of network dependencies (direct `json.Unmarshal`).

## 3. Out of Scope
- Abstraction of individual endpoints (e.g., `/liveclientdata/playerlist`) - this belongs to Feature 05.
- Logic for polling or caching the data.
- Live HTTP client changes beyond struct type updates.

## 4. Acceptance Criteria
- [x] `pkg/riotclient/models.go` exists with full, correctly-tagged structs matching `testdata/mocks/allgamedata.json`.
- [x] All nested structs are defined (abilities, champion stats, full runes, runes, items, scores, summoner spells, events, game data).
- [x] `go test ./pkg/riotclient` passes, including the new `allgamedata_test.go` assertions.
- [x] Unmarshaling succeeds even when the Riot ID fields are absent from the JSON.
- [x] `client.go` and existing `client_test.go` compile and pass after struct type changes.
- [x] Full project test suite (`go test ./...`) and Docker Compose smoke test pass.