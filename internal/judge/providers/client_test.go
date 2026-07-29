package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s, want /chat/completions", r.URL.Path)
		}
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
		if req["model"] != "test-model" {
			t.Errorf("model = %v, want test-model", req["model"])
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

	client := NewClient(server.URL, "test-key", "test-model")
	got, err := client.Complete(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Push mid and secure dragon." {
		t.Errorf("response = %s, want Push mid and secure dragon.", got)
	}
}

func TestClient_Complete_NoAuthWhenKeyEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization header should be empty, got %s", r.Header.Get("Authorization"))
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "ok"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "test-model")
	if _, err := client.Complete(context.Background(), "system", "user"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Complete_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")
	_, err := client.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("expected error for 5xx response")
	}
}

func TestClient_Complete_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model")
	client.HTTP.Timeout = 50 * time.Millisecond
	_, err := client.Complete(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestNewOpenRouterClient_Defaults(t *testing.T) {
	c := NewOpenRouterClient("key")
	if c.BaseURL != OpenRouterEndpoint {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, OpenRouterEndpoint)
	}
	if c.Model != OpenRouterDefaultModel {
		t.Errorf("Model = %q, want %q", c.Model, OpenRouterDefaultModel)
	}
	if c.APIKey != "key" {
		t.Errorf("APIKey = %q, want key", c.APIKey)
	}
}

func TestNewOpenRouterClientWithModel_EmptyModelFallsBack(t *testing.T) {
	c := NewOpenRouterClientWithModel("key", "")
	if c.Model != OpenRouterDefaultModel {
		t.Errorf("Model = %q, want default", c.Model)
	}
}

func TestNewClient_EmptyModelFallsBack(t *testing.T) {
	c := NewClient("http://localhost", "key", "")
	if c.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q, want gpt-4o-mini", c.Model)
	}
}
