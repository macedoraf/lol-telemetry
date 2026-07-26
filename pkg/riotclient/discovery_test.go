package riotclient

import (
	"testing"
	"time"
)

func TestCheckConnectionFailsWhenUnreachable(t *testing.T) {
	client := NewClientWithURL("https://127.0.0.1:1/liveclientdata")
	client.HTTPClient.Timeout = 100 * time.Millisecond
	err := client.CheckConnection()
	if err == nil {
		t.Fatal("expected error for unreachable endpoint")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestDiscoverBaseURLFailsWhenUnreachable(t *testing.T) {
	// Use a short timeout so the test runs fast.
	_, err := DiscoverBaseURL(100*time.Millisecond, false)
	if err == nil {
		t.Fatal("expected discovery to fail when no LoL API is running")
	}
}
