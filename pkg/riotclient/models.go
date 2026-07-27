package riotclient

// Ability represents a single champion ability (Q, W, E, R, or Passive).
type Ability struct {
	AbilityLevel   int    `json:"abilityLevel"`
	DisplayName    string `json:"displayName"`
	ID             string `json:"id"`
	RawDescription string `json:"rawDescription"`
	RawDisplayName string `json:"rawDisplayName"`
}

// Abilities groups the active player's abilities by slot.
type Abilities struct {
	E       Ability `json:"E"`
	Passive Ability `json:"Passive"`
	Q       Ability `json:"Q"`
	R       Ability `json:"R"`
	W       Ability `json:"W"`
}

// ChampionStats holds the active player's current combat statistics.
type ChampionStats struct {
	AbilityPower                 float64 `json:"abilityPower"`
	Armor                        float64 `json:"armor"`
	ArmorPenetrationFlat         float64 `json:"armorPenetrationFlat"`
	ArmorPenetrationPercent      float64 `json:"armorPenetrationPercent"`
	AttackDamage                 float64 `json:"attackDamage"`
	AttackRange                  float64 `json:"attackRange"`
	AttackSpeed                  float64 `json:"attackSpeed"`
	BonusArmorPenetrationPercent float64 `json:"bonusArmorPenetrationPercent"`
	BonusMagicPenetrationPercent float64 `json:"bonusMagicPenetrationPercent"`
	CooldownReduction            float64 `json:"cooldownReduction"`
	CritChance                   float64 `json:"critChance"`
	CritDamage                   float64 `json:"critDamage"`
	CurrentHealth                float64 `json:"currentHealth"`
	HealthRegenRate              float64 `json:"healthRegenRate"`
	LifeSteal                    float64 `json:"lifeSteal"`
	MagicLethality               float64 `json:"magicLethality"`
	MagicPenetrationFlat         float64 `json:"magicPenetrationFlat"`
	MagicPenetrationPercent      float64 `json:"magicPenetrationPercent"`
	MagicResist                  float64 `json:"magicResist"`
	MaxHealth                    float64 `json:"maxHealth"`
	MoveSpeed                    float64 `json:"moveSpeed"`
	PhysicalLethality            float64 `json:"physicalLethality"`
	ResourceMax                  float64 `json:"resourceMax"`
	ResourceRegenRate            float64 `json:"resourceRegenRate"`
	ResourceType                 string  `json:"resourceType"`
	ResourceValue                float64 `json:"resourceValue"`
	SpellVamp                    float64 `json:"spellVamp"`
	Tenacity                     float64 `json:"tenacity"`
}

// Rune represents a single rune (keystone, general rune, etc.).
type Rune struct {
	DisplayName    string `json:"displayName"`
	ID             int    `json:"id"`
	RawDescription string `json:"rawDescription"`
	RawDisplayName string `json:"rawDisplayName"`
}

// RuneTree represents a primary or secondary rune tree.
type RuneTree struct {
	DisplayName    string `json:"displayName"`
	ID             int    `json:"id"`
	RawDescription string `json:"rawDescription"`
	RawDisplayName string `json:"rawDisplayName"`
}

// StatRune represents a stat shard rune.
type StatRune struct {
	ID             int    `json:"id"`
	RawDescription string `json:"rawDescription"`
}

// FullRunes describes the active player's complete rune setup.
type FullRunes struct {
	GeneralRunes      []Rune     `json:"generalRunes"`
	Keystone          Rune       `json:"keystone"`
	PrimaryRuneTree   RuneTree   `json:"primaryRuneTree"`
	SecondaryRuneTree RuneTree   `json:"secondaryRuneTree"`
	StatRunes         []StatRune `json:"statRunes"`
}

// ActivePlayer represents the local player from the activePlayer block.
type ActivePlayer struct {
	Abilities     Abilities     `json:"abilities"`
	ChampionStats ChampionStats `json:"championStats"`
	CurrentGold   float64       `json:"currentGold"`
	FullRunes     FullRunes     `json:"fullRunes"`
	Level         int           `json:"level"`
	SummonerName  string        `json:"summonerName"`
}

// Item represents a single item in a player's inventory.
type Item struct {
	CanUse         bool   `json:"canUse"`
	Consumable     bool   `json:"consumable"`
	Count          int    `json:"count"`
	DisplayName    string `json:"displayName"`
	ItemID         int    `json:"itemID"`
	Price          int    `json:"price"`
	RawDescription string `json:"rawDescription"`
	RawDisplayName string `json:"rawDisplayName"`
	Slot           int    `json:"slot"`
}

// PlayerRunes represents the simplified rune information for any player.
type PlayerRunes struct {
	Keystone          RuneTree `json:"keystone"`
	PrimaryRuneTree   RuneTree `json:"primaryRuneTree"`
	SecondaryRuneTree RuneTree `json:"secondaryRuneTree"`
}

// PlayerScores holds the kill/death/assist and minion counters for a player.
type PlayerScores struct {
	Assists    int     `json:"assists"`
	CreepScore int     `json:"creepScore"`
	Deaths     int     `json:"deaths"`
	Kills      int     `json:"kills"`
	WardScore  float64 `json:"wardScore"`
}

// SummonerSpell describes a single summoner spell.
type SummonerSpell struct {
	DisplayName    string `json:"displayName"`
	RawDescription string `json:"rawDescription"`
	RawDisplayName string `json:"rawDisplayName"`
}

// SummonerSpells groups the two summoner spells for a player.
type SummonerSpells struct {
	SummonerSpellOne SummonerSpell `json:"summonerSpellOne"`
	SummonerSpellTwo SummonerSpell `json:"summonerSpellTwo"`
}

// AllPlayer represents a player in the match (active or opponent).
type AllPlayer struct {
	ChampionName    string         `json:"championName"`
	IsBot           bool           `json:"isBot"`
	IsDead          bool           `json:"isDead"`
	Items           []Item         `json:"items"`
	Level           int            `json:"level"`
	Position        string         `json:"position"`
	RawChampionName string         `json:"rawChampionName"`
	RespawnTimer    float64        `json:"respawnTimer"`
	Runes           PlayerRunes    `json:"runes"`
	Scores          PlayerScores   `json:"scores"`
	SkinID          int            `json:"skinID"`
	SummonerName    string         `json:"summonerName"`
	SummonerSpells  SummonerSpells `json:"summonerSpells"`
	Team            string         `json:"team"`
	RiotID          string         `json:"riotId"`
	RiotIDGameName  string         `json:"riotIdGameName"`
	RiotIDTagLine   string         `json:"riotIdTagLine"`
}

// Event is a single game event from the events block.
type Event struct {
	EventID   int     `json:"EventID"`
	EventName string  `json:"EventName"`
	EventTime float64 `json:"EventTime"`
}

// Events is a container for the game event list.
type Events struct {
	Events []Event `json:"Events"`
}

// GameData contains match metadata.
type GameData struct {
	GameMode   string  `json:"gameMode"`
	GameTime   float64 `json:"gameTime"`
	MapName    string  `json:"mapName"`
	MapNumber  int     `json:"mapNumber"`
	MapTerrain string  `json:"mapTerrain"`
}

// AllGameData is the root DTO returned by /allgamedata.
type AllGameData struct {
	ActivePlayer ActivePlayer `json:"activePlayer"`
	AllPlayers   []AllPlayer  `json:"allPlayers"`
	Events       Events       `json:"events"`
	GameData     GameData     `json:"gameData"`
}

// FindActivePlayer returns the player whose summoner name matches the active player.
func FindActivePlayer(data AllGameData) (AllPlayer, bool) {
	name := data.ActivePlayer.SummonerName
	for _, p := range data.AllPlayers {
		if p.SummonerName == name {
			return p, true
		}
	}
	return AllPlayer{}, false
}

// FindOpponent returns the player on the opposite team with the same position.
func FindOpponent(data AllGameData, position, activeTeam string) AllPlayer {
	if position == "" {
		return AllPlayer{}
	}
	for _, p := range data.AllPlayers {
		if p.Team != activeTeam && p.Position == position {
			return p
		}
	}
	return AllPlayer{}
}

// ItemCount returns the number of non-consumable, non-zero items a player has.
func ItemCount(items []Item) int {
	count := 0
	for _, it := range items {
		if it.ItemID != 0 && !it.Consumable {
			count++
		}
	}
	return count
}
