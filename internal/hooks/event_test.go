package hooks

import (
	"testing"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

func playerAlive(name string) riotclient.AllPlayer {
	return riotclient.AllPlayer{
		SummonerName: name,
		ChampionName: "Annie",
		Position:     "MIDDLE",
		Team:         "ORDER",
		IsDead:       false,
		Scores:       riotclient.PlayerScores{},
	}
}

func playerDead(name string) riotclient.AllPlayer {
	p := playerAlive(name)
	p.IsDead = true
	return p
}

func contextWith(prev, curr riotclient.AllGameData) types.HookContext {
	return types.HookContext{
		Data:      curr,
		PrevData:  prev,
		GameTime:  curr.GameData.GameTime,
		PrevFired: map[string]int64{},
	}
}

func dataAt(gameTime float64, players []riotclient.AllPlayer, activeGold float64, events []riotclient.Event) riotclient.AllGameData {
	name := ""
	if len(players) > 0 {
		name = players[0].SummonerName
	}
	return riotclient.AllGameData{
		ActivePlayer: riotclient.ActivePlayer{
			SummonerName:  name,
			CurrentGold:   activeGold,
			Level:         1,
			ChampionStats: riotclient.ChampionStats{},
			Abilities:     riotclient.Abilities{},
			FullRunes:     riotclient.FullRunes{},
		},
		AllPlayers: players,
		Events:     riotclient.Events{Events: events},
		GameData:   riotclient.GameData{GameTime: gameTime},
	}
}

func TestGameStartHook_ShouldFire(t *testing.T) {
	hook := GameStartHook{}
	prev := dataAt(0, []riotclient.AllPlayer{playerAlive("P")}, 0, nil)
	curr := dataAt(5, []riotclient.AllPlayer{playerAlive("P")}, 500, nil)
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected GameStartHook to fire")
	}
	if got, _ := hook.ShouldFire(contextWith(curr, curr)); got {
		t.Errorf("expected GameStartHook not to fire twice")
	}
}

func TestPlayerDeathHook_ShouldFire(t *testing.T) {
	hook := PlayerDeathHook{}
	prev := dataAt(60, []riotclient.AllPlayer{playerAlive("P")}, 500, nil)
	curr := dataAt(61, []riotclient.AllPlayer{playerDead("P")}, 500, nil)
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected PlayerDeathHook to fire")
	}
	if got, _ := hook.ShouldFire(contextWith(curr, curr)); got {
		t.Errorf("expected PlayerDeathHook not to fire while already dead")
	}
}

func TestRecallHook_ShouldFire(t *testing.T) {
	hook := RecallHook{}
	prev := dataAt(120, []riotclient.AllPlayer{playerAlive("P")}, 800, nil)
	curr := dataAt(121, []riotclient.AllPlayer{playerAlive("P")}, 1200, nil)
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected RecallHook to fire when gold crosses 1000")
	}
	prev2 := dataAt(121, []riotclient.AllPlayer{playerAlive("P")}, 1200, nil)
	if got, _ := hook.ShouldFire(contextWith(prev2, curr)); got {
		t.Errorf("expected RecallHook not to fire if already above threshold")
	}
}

func TestAllyGoldSpikeHook_ShouldFire(t *testing.T) {
	hook := AllyGoldSpikeHook{}
	prev := dataAt(120, []riotclient.AllPlayer{playerAlive("P")}, 1000, nil)
	curr := dataAt(121, []riotclient.AllPlayer{playerWithItems("P", []riotclient.Item{{ItemID: 1, Slot: 0}})}, 1000, nil)
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected AllyGoldSpikeHook to fire when item count increases")
	}
	if got, _ := hook.ShouldFire(contextWith(curr, curr)); got {
		t.Errorf("expected AllyGoldSpikeHook not to fire without item change")
	}
}

func TestEnemyGoldSpikeHook_ShouldFire(t *testing.T) {
	hook := EnemyGoldSpikeHook{}
	oppPrev := playerWithItems("O", nil)
	oppPrev.Team = "CHAOS"
	oppPrev.Position = "MIDDLE"
	oppCurr := playerWithItems("O", []riotclient.Item{{ItemID: 1, Slot: 0}})
	oppCurr.Team = "CHAOS"
	oppCurr.Position = "MIDDLE"
	prev := dataAt(120, []riotclient.AllPlayer{playerAlive("P"), oppPrev}, 1000, nil)
	curr := dataAt(121, []riotclient.AllPlayer{playerAlive("P"), oppCurr}, 1000, nil)
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected EnemyGoldSpikeHook to fire when opponent item count increases")
	}
}

func TestFirstTurretHook_ShouldFire(t *testing.T) {
	hook := FirstTurretHook{}
	prev := dataAt(300, []riotclient.AllPlayer{playerAlive("P")}, 1000, nil)
	curr := dataAt(301, []riotclient.AllPlayer{playerAlive("P")}, 1000, []riotclient.Event{{EventName: "TurretKilled", EventTime: 301}})
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected FirstTurretHook to fire")
	}
	if got, _ := hook.ShouldFire(contextWith(curr, curr)); got {
		t.Errorf("expected FirstTurretHook not to fire twice")
	}
}

func TestLaningPhaseEndHook_ShouldFire(t *testing.T) {
	hook := LaningPhaseEndHook{}
	prev := dataAt(830, []riotclient.AllPlayer{playerAlive("P")}, 1000, nil)
	curr := dataAt(840, []riotclient.AllPlayer{playerAlive("P")}, 1000, nil)
	if got, _ := hook.ShouldFire(contextWith(prev, curr)); !got {
		t.Errorf("expected LaningPhaseEndHook to fire at 14 minutes")
	}
	if got, _ := hook.ShouldFire(contextWith(curr, curr)); got {
		t.Errorf("expected LaningPhaseEndHook not to fire twice")
	}
}

func playerWithItems(name string, items []riotclient.Item) riotclient.AllPlayer {
	p := playerAlive(name)
	p.Items = items
	return p
}
