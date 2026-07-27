package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecorder_JSONLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("20260727-120000-abcdef"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	raw := json.RawMessage(`{"activePlayer":{"summonerName":"Player"},"allPlayers":[],"events":{"Events":[]},"gameData":{"gameTime":123.4,"gameMode":"CLASSIC"}}`)
	for i := 0; i < 5; i++ {
		r.Record(TelemetryRecord{
			Version:  1,
			Type:     "telemetry",
			Ts:       time.Now().UnixMilli(),
			Session:  "20260727-120000-abcdef",
			GameTime: 123.4 + float64(i),
			Data:     raw,
		})
	}

	// Close flushes everything.
	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "20260727-120000-abcdef", "telemetry.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open telemetry.jsonl: %v", err)
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var rec TelemetryRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		if rec.Version != 1 || rec.Type != "telemetry" || rec.Session != "20260727-120000-abcdef" {
			t.Errorf("unexpected record: %+v", rec)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 5 {
		t.Errorf("read %d lines, want 5", count)
	}
}

func TestRecorder_DropWhenChannelFull(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("session-drop"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	// Block the background writer so the channel cannot drain while we flood
	// it. Without this the loop may consume records fast enough to keep the
	// channel empty, making the drop count non-deterministic.
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()

	// Flood the channel.
	done := make(chan struct{})
	go func() {
		for i := 0; i < chanCapacity+50; i++ {
			r.Record(TelemetryRecord{Version: 1, Type: "telemetry", Session: "session-drop", Data: json.RawMessage(`{}`)})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Record blocked on a full channel")
	}

	if r.Dropped() == 0 {
		t.Errorf("expected dropped records, got 0")
	}
}

func TestRecorder_CloseDrainsChannel(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := r.StartSession("session-drain"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	for i := 0; i < 10; i++ {
		r.Record(TelemetryRecord{Version: 1, Type: "telemetry", Session: "session-drain", Data: json.RawMessage(`{}`)})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if r.Written() != 10 {
		t.Errorf("written = %d, want 10", r.Written())
	}
}

func TestRecorder_SessionDirectoryLayout(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("sess-1"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	r.Record(TelemetryRecord{Version: 1, Type: "telemetry", Session: "sess-1", Data: json.RawMessage(`{}`)})

	// Wait for the background goroutine to write and flush.
	time.Sleep(100 * time.Millisecond)
	r.EndSession()
	time.Sleep(100 * time.Millisecond)

	path := filepath.Join(dir, "sess-1", "telemetry.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}

	// Starting a new session should not delete the old file.
	if err := r.StartSession("sess-2"); err != nil {
		t.Fatalf("StartSession sess-2: %v", err)
	}
	if r.SessionID() != "sess-2" {
		t.Errorf("SessionID = %q, want sess-2", r.SessionID())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sess-1 file: %v", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		t.Errorf("sess-1 file is empty after starting sess-2")
	}
}
