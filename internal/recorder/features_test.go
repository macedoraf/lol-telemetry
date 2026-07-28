package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lol-telemetry/internal/types"
)

func TestRecorder_FeatureRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("session-features"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	fv := types.FeatureVector{
		GameMinute: 10,
		WindowSec:  12.4,
		Samples:    1,
		Player: types.PlayerFeatures{
			GoldPerMin: 350.5,
			Level:      9,
		},
	}
	r.RecordFeature(FeatureRecord{
		Version:    1,
		Type:       "features",
		Ts:         time.Now().UnixMilli(),
		GameTime:   612.4,
		GameMinute: 10,
		Features:   fv,
	})

	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "session-features", "features.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open features.jsonl: %v", err)
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var rec FeatureRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		if rec.Type != "features" || rec.Session != "session-features" {
			t.Errorf("unexpected record: %+v", rec)
		}
		if rec.Features.Player.GoldPerMin != 350.5 {
			t.Errorf("GoldPerMin = %f, want 350.5", rec.Features.Player.GoldPerMin)
		}
		count++
	}
	if count != 1 {
		t.Errorf("read %d lines, want 1", count)
	}
}

func TestRecorder_NoFeaturesFileWithoutFeatures(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("session-no-features"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "session-no-features", "features.jsonl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("features.jsonl should not exist when no features were recorded")
	}
}
