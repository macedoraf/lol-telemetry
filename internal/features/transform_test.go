package features

import (
	"testing"

	"lol-telemetry/pkg/riotclient"
)

func testWindow(samples []Sample, events []riotclient.Event) Window {
	return &window{samples: samples, events: events}
}

func player(name, team, pos string, active bool, level, cs, items, kills, deaths int, gold float64, dead bool, respawn float64) PlayerSample {
	return PlayerSample{
		SummonerName:   name,
		ChampionName:   name,
		Team:           team,
		Position:       pos,
		Level:          level,
		CS:             cs,
		ItemsCompleted: items,
		ItemsGold:      items * 1000,
		Kills:          kills,
		Deaths:         deaths,
		IsActive:       active,
		Gold:           gold,
		IsDead:         dead,
		RespawnTimer:   respawn,
	}
}

func TestGoldTransformer(t *testing.T) {
	p1 := player("P1", "ORDER", "MIDDLE", true, 5, 60, 2, 1, 0, 500, false, 0)
	samples := []Sample{
		{GameTime: 120, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
		{GameTime: 180, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
		{GameTime: 240, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
	}
	samples[0].Players[0].Gold = 800
	samples[1].Players[0].Gold = 1500
	samples[2].Players[0].Gold = 2400

	fv := NewPipeline().Compute(testWindow(samples, nil))

	wantPerMin := (2400.0 - 500.0) / (240.0 / 60.0)
	if fv.Player.GoldPerMin != wantPerMin {
		t.Errorf("GoldPerMin = %f, want %f", fv.Player.GoldPerMin, wantPerMin)
	}
	if fv.Player.GoldDelta1m != 900 {
		t.Errorf("GoldDelta1m = %f, want 900", fv.Player.GoldDelta1m)
	}
	if fv.Player.GoldDelta5m != 1600 {
		t.Errorf("GoldDelta5m = %f, want 1600", fv.Player.GoldDelta5m)
	}
	if fv.Player.GoldSpike1m {
		t.Errorf("GoldSpike1m = true, want false")
	}

	// ARAM starting gold
	samples[0].GameMode = "ARAM"
	samples[1].GameMode = "ARAM"
	samples[2].GameMode = "ARAM"
	fv2 := NewPipeline().Compute(testWindow(samples, nil))
	wantPerMin2 := (2400.0 - 1400.0) / (240.0 / 60.0)
	if fv2.Player.GoldPerMin != wantPerMin2 {
		t.Errorf("ARAM GoldPerMin = %f, want %f", fv2.Player.GoldPerMin, wantPerMin2)
	}

	// Gold spike
	samples2 := []Sample{
		{GameTime: 180, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
		{GameTime: 240, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
		{GameTime: 300, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
	}
	samples2[0].Players[0].Gold = 2400
	samples2[1].Players[0].Gold = 2400
	samples2[2].Players[0].Gold = 3800
	fv3 := NewPipeline().Compute(testWindow(samples2, nil))
	if fv3.Player.GoldDelta1m != 1400 {
		t.Errorf("GoldDelta1m = %f, want 1400", fv3.Player.GoldDelta1m)
	}
	if !fv3.Player.GoldSpike1m {
		t.Errorf("GoldSpike1m = false, want true")
	}
}

func TestGoldTransformer_ColdStart(t *testing.T) {
	p1 := player("P1", "ORDER", "MIDDLE", true, 1, 0, 0, 0, 0, 500, false, 0)
	samples := []Sample{
		{GameTime: 0.5, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
	}
	fv := NewPipeline().Compute(testWindow(samples, nil))
	if fv.Player.GoldPerMin != 0 {
		t.Errorf("GoldPerMin at t=0.5 = %f, want 0", fv.Player.GoldPerMin)
	}
}

func TestXPTransformer(t *testing.T) {
	p1 := player("P1", "ORDER", "MIDDLE", true, 5, 60, 1, 0, 0, 1000, false, 0)
	samples := []Sample{
		{GameTime: 120, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
	}
	fv := NewPipeline().Compute(testWindow(samples, nil))
	want := float64(XP_TABLE[5]) / (120.0 / 60.0)
	if fv.Player.XPPerMin != want {
		t.Errorf("XPPerMin = %f, want %f", fv.Player.XPPerMin, want)
	}
	if fv.Team.AvgXPPerMin != want {
		t.Errorf("AvgXPPerMin = %f, want %f", fv.Team.AvgXPPerMin, want)
	}
}

func TestXPTransformer_LevelOne(t *testing.T) {
	p1 := player("P1", "ORDER", "MIDDLE", true, 1, 0, 0, 0, 0, 500, false, 0)
	samples := []Sample{
		{GameTime: 0.5, GameMode: "CLASSIC", Players: []PlayerSample{p1}},
	}
	fv := NewPipeline().Compute(testWindow(samples, nil))
	if fv.Player.XPPerMin != 0 {
		t.Errorf("XPPerMin at t=0.5 level 1 = %f, want 0", fv.Player.XPPerMin)
	}
}

func TestSpikesTransformer(t *testing.T) {
	first := []PlayerSample{
		player("P1", "ORDER", "MIDDLE", true, 5, 60, 1, 1, 0, 1000, false, 0),
		player("P2", "CHAOS", "MIDDLE", false, 5, 50, 1, 0, 1, 0, false, 0),
	}
	last := []PlayerSample{
		player("P1", "ORDER", "MIDDLE", true, 5, 80, 2, 1, 0, 1200, false, 0),
		player("P2", "CHAOS", "MIDDLE", false, 6, 65, 1, 0, 1, 0, false, 0),
	}
	w := testWindow([]Sample{
		{GameTime: 60, GameMode: "CLASSIC", Players: first},
		{GameTime: 120, GameMode: "CLASSIC", Players: last},
	}, nil)
	fv := NewPipeline().Compute(w)

	if fv.Team.ItemCompletions1m != 1 {
		t.Errorf("ally itemCompletions1m = %d, want 1", fv.Team.ItemCompletions1m)
	}
	if fv.Enemy.LevelUps1m != 1 {
		t.Errorf("enemy levelUps1m = %d, want 1", fv.Enemy.LevelUps1m)
	}
	if len(fv.Team.Spikes) != 1 {
		t.Errorf("ally spikes = %d, want 1", len(fv.Team.Spikes))
	}
	if len(fv.Enemy.Spikes) != 1 {
		t.Errorf("enemy spikes = %d, want 1", len(fv.Enemy.Spikes))
	}
}

func TestMatchupTransformer(t *testing.T) {
	players := []PlayerSample{
		player("P1", "ORDER", "MIDDLE", true, 5, 60, 2, 1, 0, 1000, false, 0),
		player("P2", "CHAOS", "MIDDLE", false, 6, 70, 1, 0, 1, 0, false, 0),
	}
	fv := NewPipeline().Compute(testWindow([]Sample{
		{GameTime: 120, GameMode: "CLASSIC", Players: players},
	}, nil))
	if fv.Matchup == nil {
		t.Fatal("expected matchup")
	}
	if fv.Matchup.LevelDiff != -1 {
		t.Errorf("LevelDiff = %d, want -1", fv.Matchup.LevelDiff)
	}
	if fv.Matchup.CSDiff != -10 {
		t.Errorf("CSDiff = %d, want -10", fv.Matchup.CSDiff)
	}
	if fv.Matchup.ItemDiff != 1 {
		t.Errorf("ItemDiff = %d, want 1", fv.Matchup.ItemDiff)
	}
	if fv.Matchup.KillDiff != 1 {
		t.Errorf("KillDiff = %d, want 1", fv.Matchup.KillDiff)
	}
}

func TestMatchupTransformer_NoPosition(t *testing.T) {
	players := []PlayerSample{
		player("P1", "ORDER", "", true, 5, 60, 2, 1, 0, 1000, false, 0),
		player("P2", "CHAOS", "", false, 6, 70, 1, 0, 1, 0, false, 0),
	}
	fv := NewPipeline().Compute(testWindow([]Sample{
		{GameTime: 120, GameMode: "ARAM", Players: players},
	}, nil))
	if fv.Matchup != nil {
		t.Errorf("expected nil matchup for ARAM/empty position, got %+v", fv.Matchup)
	}
}

func TestObjectivesTransformer(t *testing.T) {
	players := []PlayerSample{
		player("P1", "ORDER", "MIDDLE", true, 9, 82, 2, 1, 2, 4500, false, 0),
		player("P2", "CHAOS", "MIDDLE", false, 10, 95, 1, 3, 1, 0, true, 28.5),
	}
	events := []riotclient.Event{
		{EventID: 0, EventName: "GameStart", EventTime: 0},
		{EventID: 1, EventName: "FirstBrick", EventTime: 301, KillerName: "P2"},
		{EventID: 2, EventName: "DragonKill", EventTime: 312, DragonType: "Infernal", Stolen: "True", KillerName: "P2", Assisters: []string{"P2"}},
		{EventID: 3, EventName: "TurretKilled", EventTime: 315, TurretKilled: "Turret_T1_C_05_A", KillerName: "P2", Assisters: []string{"P2"}},
		{EventID: 4, EventName: "ChampionKill", EventTime: 580, KillerName: "P2", VictimName: "P1", Assisters: []string{"P2"}},
		{EventID: 5, EventName: "Ace", EventTime: 590, AcingTeam: "CHAOS"},
	}
	fv := NewPipeline().Compute(testWindow([]Sample{
		{GameTime: 612, GameMode: "CLASSIC", Players: players},
	}, events))

	if fv.Enemy.Objectives.Dragons != 1 {
		t.Errorf("enemy dragons = %d, want 1", fv.Enemy.Objectives.Dragons)
	}
	if fv.Enemy.Objectives.Steals != 1 {
		t.Errorf("enemy steals = %d, want 1", fv.Enemy.Objectives.Steals)
	}
	if fv.Enemy.Objectives.Towers != 1 {
		t.Errorf("enemy towers = %d, want 1", fv.Enemy.Objectives.Towers)
	}
	if fv.Enemy.Kills1m != 1 {
		t.Errorf("enemy kills1m = %d, want 1", fv.Enemy.Kills1m)
	}
	if fv.Enemy.DeadNow != 1 {
		t.Errorf("enemy deadNow = %d, want 1", fv.Enemy.DeadNow)
	}
	if fv.Enemy.MaxRespawnSec != 28.5 {
		t.Errorf("enemy MaxRespawnSec = %f, want 28.5", fv.Enemy.MaxRespawnSec)
	}
	if fv.Team.Objectives.Dragons != 0 || fv.Team.Objectives.Towers != 0 {
		t.Errorf("ally objectives should be zero, got %+v", fv.Team.Objectives)
	}
	if !contains(fv.Enemy.Spikes, "Ace by CHAOS @09:50") {
		t.Errorf("missing ace spike, got %v", fv.Enemy.Spikes)
	}
	if !contains(fv.Enemy.Spikes, "Infernal dragon stolen by CHAOS @05:12") {
		t.Errorf("missing dragon steal spike, got %v", fv.Enemy.Spikes)
	}
}

func contains(sl []string, v string) bool {
	for _, s := range sl {
		if s == v {
			return true
		}
	}
	return false
}

func TestObjectivesTransformer_Kills1mWindow(t *testing.T) {
	players := []PlayerSample{
		player("P1", "ORDER", "MIDDLE", true, 9, 82, 2, 1, 2, 4500, false, 0),
		player("P2", "CHAOS", "MIDDLE", false, 10, 95, 1, 3, 1, 0, false, 0),
	}
	events := []riotclient.Event{
		{EventID: 1, EventName: "ChampionKill", EventTime: 100, KillerName: "P2"},
		{EventID: 2, EventName: "ChampionKill", EventTime: 580, KillerName: "P2"},
	}
	fv := NewPipeline().Compute(testWindow([]Sample{
		{GameTime: 612, GameMode: "CLASSIC", Players: players},
	}, events))
	if fv.Enemy.Kills1m != 1 {
		t.Errorf("enemy kills1m = %d, want 1", fv.Enemy.Kills1m)
	}
}
