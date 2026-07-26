package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %s, want Bearer test-key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header = %s, want application/json", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if req["model"] != "openai/gpt-4o-mini" {
			t.Errorf("model = %v, want openai/gpt-4o-mini", req["model"])
		}
		messages, ok := req["messages"].([]any)
		if !ok || len(messages) != 2 {
			t.Fatalf("expected 2 messages, got %v", messages)
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "Push mid and secure dragon.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.Endpoint = server.URL

	got, err := client.Complete(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Push mid and secure dragon." {
		t.Errorf("response = %s, want Push mid and secure dragon.", got)
	}
}

func TestClient_Complete_CustomModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if req["model"] != "anthropic/claude-3.5-sonnet" {
			t.Errorf("model = %v, want anthropic/claude-3.5-sonnet", req["model"])
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "Custom model response.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithModel("test-key", "anthropic/claude-3.5-sonnet")
	client.Endpoint = server.URL

	got, err := client.Complete(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Custom model response." {
		t.Errorf("response = %s, want Custom model response.", got)
	}
}

func TestClient_Complete_EmptyModelFallsBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if req["model"] != DefaultModel {
			t.Errorf("model = %v, want default %s", req["model"], DefaultModel)
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "Fallback model response.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithModel("test-key", "")
	client.Endpoint = server.URL

	got, err := client.Complete(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Fallback model response." {
		t.Errorf("response = %s, want Fallback model response.", got)
	}
}

func TestClient_Complete_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.Endpoint = server.URL

	_, err := client.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("expected error for 5xx response")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %v, want to contain unexpected status 500", err)
	}
}

func TestClient_Complete_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.Endpoint = server.URL
	client.HTTP.Timeout = 50 * time.Millisecond

	_, err := client.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
