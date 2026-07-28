package features

import (
	"sync"
	"testing"

	"lol-telemetry/pkg/riotclient"
)

func TestTracker_AddExtractsSample(t *testing.T) {
	data := makeAllGameData(120.0, "CLASSIC", []PlayerSample{
		{SummonerName: "P1", ChampionName: "Annie", Team: "ORDER", Position: "MIDDLE", Level: 5, Kills: 1, Deaths: 0, Assists: 2, CS: 60, ItemsCompleted: 2, ItemsGold: 1500, IsActive: true, IsDead: false, RespawnTimer: 0, Gold: 1700},
		{SummonerName: "P2", ChampionName: "Ahri", Team: "CHAOS", Position: "MIDDLE", Level: 6, Kills: 0, Deaths: 1, Assists: 0, CS: 70, ItemsCompleted: 1, ItemsGold: 1100, IsActive: false, IsDead: true, RespawnTimer: 28.5, Gold: 0},
	})
	data.Events.Events = []riotclient.Event{
		{EventID: 0, EventName: "GameStart", EventTime: 0},
		{EventID: 1, EventName: "DragonKill", EventTime: 60, DragonType: "Infernal", Stolen: "True", KillerName: "P2"},
	}

	tr := NewTracker()
	tr.Add(data)

	w := tr.Window()
	last, ok := w.Last()
	if !ok {
		t.Fatal("expected last sample")
	}
	if last.GameTime != 120 {
		t.Errorf("GameTime = %f, want 120", last.GameTime)
	}
	if last.GameMode != "CLASSIC" {
		t.Errorf("GameMode = %q, want CLASSIC", last.GameMode)
	}
	if len(last.Players) != 2 {
		t.Fatalf("Players = %d, want 2", len(last.Players))
	}

	p := last.Players[0]
	if !p.IsActive {
		t.Errorf("P1 IsActive = false")
	}
	if p.Gold != 1700 {
		t.Errorf("P1 Gold = %f, want 1700", p.Gold)
	}
	if p.ItemsGold != 1500 {
		t.Errorf("P1 ItemsGold = %d, want 1500", p.ItemsGold)
	}

	op := last.Players[1]
	if !op.IsDead || op.RespawnTimer != 28.5 {
		t.Errorf("P2 death state mismatch: isDead=%v respawn=%f", op.IsDead, op.RespawnTimer)
	}

	evts := w.Events()
	if len(evts) != 2 {
		t.Fatalf("Events = %d, want 2", len(evts))
	}
	if evts[1].DragonType != "Infernal" {
		t.Errorf("DragonType = %q, want Infernal", evts[1].DragonType)
	}
}

func TestTracker_RingWrap(t *testing.T) {
	tr := NewTracker()
	for i := 0; i < 3601; i++ {
		tr.Add(makeAllGameData(float64(i), "CLASSIC", []PlayerSample{
			{SummonerName: "P1", Team: "ORDER", IsActive: true, Gold: float64(i * 10)},
		}))
	}
	samples := tr.Window().Samples()
	if len(samples) != 3600 {
		t.Fatalf("len(samples) = %d, want 3600", len(samples))
	}
	last := samples[len(samples)-1]
	if last.GameTime != 3600 {
		t.Errorf("last GameTime = %f, want 3600", last.GameTime)
	}
	if last.Players[0].Gold != 36000 {
		t.Errorf("last Gold = %f, want 36000", last.Players[0].Gold)
	}
}

func TestTracker_Since(t *testing.T) {
	tr := NewTracker()
	for _, gt := range []float64{10, 70, 130, 200} {
		tr.Add(makeAllGameData(gt, "CLASSIC", []PlayerSample{
			{SummonerName: "P1", Team: "ORDER", IsActive: true, Gold: gt},
		}))
	}

	w := tr.Window()
	if got := len(w.Since(60)); got != 1 {
		t.Errorf("Since(60) = %d samples, want 1 (only t=200)", got)
	}
	if got := len(w.Since(90)); got != 2 {
		t.Errorf("Since(90) = %d samples, want 2 (t=130,200)", got)
	}
	if got := len(w.Since(180)); got != 3 {
		t.Errorf("Since(180) = %d samples, want 3 (t=70,130,200)", got)
	}
}

func TestTracker_EventsDeepCopy(t *testing.T) {
	tr := NewTracker()
	tr.Add(makeAllGameData(10, "CLASSIC", []PlayerSample{
		{SummonerName: "P1", Team: "ORDER", IsActive: true, Gold: 500},
	}))
	tr.Add(makeAllGameData(20, "CLASSIC", []PlayerSample{
		{SummonerName: "P1", Team: "ORDER", IsActive: true, Gold: 600},
	}))

	evts := tr.Window().Events()
	if len(evts) != 1 {
		t.Fatalf("events = %d, want 1", len(evts))
	}
	evts[0].EventName = "Mutated"

	evts2 := tr.Window().Events()
	if evts2[0].EventName != "GameStart" {
		t.Errorf("tracker event was mutated through returned copy")
	}
}

func TestTracker_Reset(t *testing.T) {
	tr := NewTracker()
	tr.Add(makeAllGameData(10, "CLASSIC", []PlayerSample{
		{SummonerName: "P1", Team: "ORDER", IsActive: true, Gold: 500},
	}))
	tr.Reset()
	if _, ok := tr.Window().Last(); ok {
		t.Error("expected empty window after Reset")
	}
	if evts := tr.Window().Events(); len(evts) != 0 {
		t.Errorf("expected no events after Reset, got %d", len(evts))
	}
}

func TestTracker_ConcurrentAddAndWindow(t *testing.T) {
	tr := NewTracker()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			tr.Add(makeAllGameData(float64(i), "CLASSIC", []PlayerSample{
				{SummonerName: "P1", Team: "ORDER", IsActive: true, Gold: float64(i)},
			}))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = tr.Window().Samples()
			_ = tr.Window().Events()
		}
	}()
	wg.Wait()
}

func makeAllGameData(gameTime float64, mode string, players []PlayerSample) riotclient.AllGameData {
	var active riotclient.ActivePlayer
	var allPlayers []riotclient.AllPlayer
	for _, p := range players {
		if p.IsActive {
			active = riotclient.ActivePlayer{
				SummonerName: p.SummonerName,
				CurrentGold:  p.Gold,
				Level:        p.Level,
			}
		}
		items := makeItems(p.ItemsCompleted, p.ItemsGold)
		allPlayers = append(allPlayers, riotclient.AllPlayer{
			SummonerName: p.SummonerName,
			ChampionName: p.ChampionName,
			Team:         p.Team,
			Position:     p.Position,
			Level:        p.Level,
			IsDead:       p.IsDead,
			RespawnTimer: p.RespawnTimer,
			Items:        items,
			Scores: riotclient.PlayerScores{
				Kills:      p.Kills,
				Deaths:     p.Deaths,
				Assists:    p.Assists,
				CreepScore: p.CS,
			},
		})
	}
	return riotclient.AllGameData{
		ActivePlayer: active,
		AllPlayers:   allPlayers,
		Events:       riotclient.Events{Events: []riotclient.Event{{EventID: 0, EventName: "GameStart", EventTime: 0}}},
		GameData:     riotclient.GameData{GameTime: gameTime, GameMode: mode},
	}
}

func makeItems(count, totalGold int) []riotclient.Item {
	if count <= 0 || totalGold <= 0 {
		return nil
	}
	price := totalGold / count
	items := make([]riotclient.Item, count)
	for i := range items {
		items[i] = riotclient.Item{
			ItemID:     1000 + i,
			Consumable: false,
			Price:      price,
		}
	}
	return items
}
