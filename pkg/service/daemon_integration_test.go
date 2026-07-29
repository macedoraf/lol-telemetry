package service

import (
	"bufio"
	"context"
	"encoding/json"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lol-telemetry/internal/judge"
	"lol-telemetry/internal/orchestrator"
	"lol-telemetry/pkg/riotclient"
)

type fakeLLM struct{}

func (fakeLLM) Complete(ctx context.Context, systemPrompt, prompt string) (string, error) {
	return `{"advice":"Recall","reasoning":"Low health"}`, nil
}

func TestDaemon_RecordingCreatesJSONL(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/allgamedata" && r.URL.Path != "/activeplayername" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/activeplayername" {
			_, _ = w.Write([]byte(`"Player"`))
			return
		}
		_, _ = w.Write([]byte(`{
			"activePlayer": {"summonerName":"Player","level":1,"currentGold":0,"abilities":{"E":{},"Passive":{},"Q":{},"R":{},"W":{}},"championStats":{},"fullRunes":{"generalRunes":[],"keystone":{},"primaryRuneTree":{},"secondaryRuneTree":{},"statRunes":[]}},
			"allPlayers": [],
			"events": {"Events": []},
			"gameData": {"gameMode":"CLASSIC","gameTime":12.5,"mapName":"SR","mapNumber":11,"mapTerrain":"Default"}
		}`))
	}))
	defer server.Close()

	cfg := DaemonConfig{
		Port:          "0",
		BaseURL:       server.URL,
		PollInterval:  100 * time.Millisecond,
		JudgeEnabled:  false,
		RecordEnabled: true,
		RecordingsDir: dir,
	}
	d := NewDaemon(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := d.Run(ctx); err != nil && err != context.Canceled {
			t.Errorf("Run error: %v", err)
		}
	}()

	// Wait for a few polls and a session to start.
	time.Sleep(350 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read recordings dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no session directory created")
	}

	sessionDir := filepath.Join(dir, entries[0].Name())
	path := filepath.Join(sessionDir, "telemetry.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open telemetry.jsonl: %v", err)
	}
	defer f.Close()

	var lines int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec struct {
			Version  int             `json:"v"`
			Type     string          `json:"type"`
			Session  string          `json:"session"`
			GameTime float64         `json:"gameTime"`
			Data     json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			t.Fatalf("invalid JSONL line: %v", err)
		}
		if rec.Version != 1 || rec.Type != "telemetry" {
			t.Errorf("unexpected envelope: %+v", rec)
		}
		if rec.Session == "" {
			t.Errorf("empty session id")
		}
		if rec.GameTime != 12.5 {
			t.Errorf("gameTime = %f, want 12.5", rec.GameTime)
		}
		var data riotclient.AllGameData
		if err := json.Unmarshal(rec.Data, &data); err != nil {
			t.Fatalf("data is not valid AllGameData: %v", err)
		}
		lines++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if lines == 0 {
		t.Errorf("no telemetry lines written")
	}
}

func TestDaemon_TipRecordingCorrelatesWithTelemetry(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/activeplayername" {
			_, _ = w.Write([]byte(`"Player"`))
			return
		}
		_, _ = w.Write([]byte(`{
			"activePlayer": {"summonerName":"Player","level":10,"currentGold":3000,"abilities":{"E":{},"Passive":{},"Q":{},"R":{},"W":{}},"championStats":{},"fullRunes":{"generalRunes":[],"keystone":{},"primaryRuneTree":{},"secondaryRuneTree":{},"statRunes":[]}},
			"allPlayers": [
				{"summonerName":"Player","championName":"Annie","level":10,"position":"MIDDLE","team":"ORDER","scores":{"kills":1,"deaths":0,"assists":2,"creepScore":80,"wardScore":0},"items":[],"runes":{"keystone":{},"primaryRuneTree":{},"secondaryRuneTree":{}},"summonerSpells":{}}
			],
			"events": {"Events": []},
			"gameData": {"gameMode":"CLASSIC","gameTime":300,"mapName":"SR","mapNumber":11,"mapTerrain":"Default"}
		}`))
	}))
	defer server.Close()

	cfg := DaemonConfig{
		Port:          "0",
		BaseURL:       server.URL,
		PollInterval:  100 * time.Millisecond,
		JudgeEnabled:  true,
		OpenRouterKey: "fake-key",
		RecordEnabled: true,
		RecordingsDir: dir,
	}
	d := NewDaemon(cfg)
	// Inject a fake LLM so the judge produces a tip.
	if d.orch != nil {
		// Orchestrator already created without a real judge because no OpenRouter key;
		// rebuild with fake judge.
		reg := d.reg
		builder := d.builder
		d.orch = orchestrator.NewOrchestrator(d.client, reg, builder, judge.NewJudge(fakeLLM{}), nil, nil)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := d.Run(ctx); err != nil && err != context.Canceled {
			t.Errorf("Run error: %v", err)
		}
	}()

	// Wait for the periodic hook to fire at 300s game time.
	time.Sleep(500 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read recordings dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no session directory created")
	}

	sessionDir := filepath.Join(dir, entries[0].Name())
	tipPath := filepath.Join(sessionDir, "tips.jsonl")
	tipsData, err := os.ReadFile(tipPath)
	if err != nil {
		t.Fatalf("read tips.jsonl: %v", err)
	}

	var tip struct {
		Session  string  `json:"session"`
		GameTime float64 `json:"gameTime"`
		Advice   string  `json:"advice"`
	}
	if err := json.Unmarshal(bytes.Split(tipsData, []byte("\n"))[0], &tip); err != nil {
		t.Fatalf("invalid tip JSON: %v", err)
	}
	if tip.Session == "" {
		t.Fatalf("tip has no session")
	}
	if tip.Advice != "Recall" {
		t.Errorf("advice = %q, want Recall", tip.Advice)
	}

	// Find a telemetry line within poll interval of the tip's gameTime.
	telePath := filepath.Join(sessionDir, "telemetry.jsonl")
	teleFile, err := os.Open(telePath)
	if err != nil {
		t.Fatalf("open telemetry.jsonl: %v", err)
	}
	defer teleFile.Close()

	var found bool
	scanner := bufio.NewScanner(teleFile)
	for scanner.Scan() {
		var rec struct {
			Session  string  `json:"session"`
			GameTime float64 `json:"gameTime"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.Session == tip.Session && abs(rec.GameTime-tip.GameTime) < 0.15 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no telemetry line correlated with tip (session=%s gameTime=%f)", tip.Session, tip.GameTime)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestDaemon_RecordingDisabledCreatesNoFiles(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/activeplayername" {
			_, _ = w.Write([]byte(`"Player"`))
			return
		}
		_, _ = w.Write([]byte(`{
			"activePlayer": {"summonerName":"Player","level":1,"currentGold":0,"abilities":{"E":{},"Passive":{},"Q":{},"R":{},"W":{}},"championStats":{},"fullRunes":{"generalRunes":[],"keystone":{},"primaryRuneTree":{},"secondaryRuneTree":{},"statRunes":[]}},
			"allPlayers": [],
			"events": {"Events": []},
			"gameData": {"gameMode":"CLASSIC","gameTime":5.0,"mapName":"SR","mapNumber":11,"mapTerrain":"Default"}
		}`))
	}))
	defer server.Close()

	cfg := DaemonConfig{
		Port:          "0",
		BaseURL:       server.URL,
		PollInterval:  100 * time.Millisecond,
		JudgeEnabled:  false,
		RecordEnabled: false,
		RecordingsDir: dir,
	}
	d := NewDaemon(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := d.Run(ctx); err != nil && err != context.Canceled {
			t.Errorf("Run error: %v", err)
		}
	}()

	time.Sleep(250 * time.Millisecond)
	cancel()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read recordings dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no recordings, found %d entries", len(entries))
	}
}
