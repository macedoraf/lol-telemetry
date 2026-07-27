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

func TestRecorder_TipRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("session-tips"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	r.RecordTip(TipRecord{
		Version:    1,
		Type:       "tip",
		Ts:         time.Now().UnixMilli(),
		GameTime:   123.4,
		GameMinute: 2,
		HookName:   "periodic-5min",
		Question:   "q",
		Advice:     "Recall.",
		Reasoning:  "Because.",
	})

	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "session-tips", "tips.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open tips.jsonl: %v", err)
	}
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var rec TipRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		if rec.Type != "tip" || rec.HookName != "periodic-5min" || rec.Session != "session-tips" {
			t.Errorf("unexpected record: %+v", rec)
		}
		count++
	}
	if count != 1 {
		t.Errorf("read %d lines, want 1", count)
	}
}

func TestRecorder_TipsOnlyWhenSessionActive(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	// No session started: tip is accepted into the channel but dropped on write.
	r.RecordTip(TipRecord{Version: 1, Type: "tip", HookName: "x", Advice: "a"})
	time.Sleep(50 * time.Millisecond)

	if r.TipWritten() != 0 {
		t.Errorf("TipWritten = %d, want 0", r.TipWritten())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no session dir, got %d", len(entries))
	}
}

func TestRecorder_NoTipsFileWithoutTips(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("session-no-tips"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "session-no-tips", "tips.jsonl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("tips.jsonl should not exist when no tips were recorded")
	}
}

func TestRecorder_TipDropWhenChannelFull(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(context.Background())

	if err := r.StartSession("session-drop-tips"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < chanCapacity+50; i++ {
			r.RecordTip(TipRecord{Version: 1, Type: "tip", HookName: "x", Advice: "a"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RecordTip blocked on a full channel")
	}

	if r.TipDropped() == 0 {
		t.Errorf("expected dropped tips, got 0")
	}
}

func TestRecorder_TipDrainOnClose(t *testing.T) {
	dir := t.TempDir()
	r, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := r.StartSession("session-drain-tips"); err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	for i := 0; i < 10; i++ {
		r.RecordTip(TipRecord{Version: 1, Type: "tip", HookName: "x", Advice: "a"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if r.TipWritten() != 10 {
		t.Errorf("TipWritten = %d, want 10", r.TipWritten())
	}

	path := filepath.Join(dir, "session-drain-tips", "tips.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tips.jsonl: %v", err)
	}
	if strings.Count(string(data), "\n") != 10 {
		t.Errorf("expected 10 lines, got %d", strings.Count(string(data), "\n"))
	}
}
