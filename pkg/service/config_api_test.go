package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/orchestrator"
	"lol-telemetry/pkg/riotclient"
)

func setupRuntimeConfig() *RuntimeConfig {
	client := riotclient.NewClient()
	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	builder := payload.NewBuilder("en")
	orch := orchestrator.NewOrchestrator(client, reg, builder, nil, nil, nil)
	return NewRuntimeConfig("en", reg, builder, orch)
}

func TestHandlePatchConfig_Prompt(t *testing.T) {
	rc := setupRuntimeConfig()
	server := httptest.NewServer(http.HandlerFunc(handlePatchConfig(rc)))
	defer server.Close()

	override := "Focus only on dragon control."
	body, _ := json.Marshal(ConfigPatch{Judge: &JudgeConfigPatch{Prompt: &override}})
	resp, err := http.DefaultClient.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var view ConfigView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Judge.PromptOverride != override {
		t.Errorf("PromptOverride = %q, want %q", view.Judge.PromptOverride, override)
	}
	if !strings.Contains(view.Judge.EffectivePrompt, override) {
		t.Errorf("EffectivePrompt missing override: %s", view.Judge.EffectivePrompt)
	}
	if !strings.Contains(view.Judge.EffectivePrompt, "Respond entirely in English") {
		t.Errorf("EffectivePrompt missing language directive")
	}
}

func TestHandlePatchConfig_PromptInvalid(t *testing.T) {
	rc := setupRuntimeConfig()
	server := httptest.NewServer(http.HandlerFunc(handlePatchConfig(rc)))
	defer server.Close()

	// Set a valid override first.
	override := "Valid override."
	body, _ := json.Marshal(ConfigPatch{Judge: &JudgeConfigPatch{Prompt: &override}})
	resp, err := http.DefaultClient.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	resp.Body.Close()

	// Then send an invalid one.
	bad := strings.Repeat("x", 4001)
	body, _ = json.Marshal(ConfigPatch{Judge: &JudgeConfigPatch{Prompt: &bad}})
	resp, err = http.DefaultClient.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	// Ensure previous override remains.
	snap := rc.Snapshot()
	if snap.Judge.PromptOverride != override {
		t.Errorf("previous override changed to %q", snap.Judge.PromptOverride)
	}
}

func TestHandlePatchConfig_PromptReset(t *testing.T) {
	rc := setupRuntimeConfig()
	server := httptest.NewServer(http.HandlerFunc(handlePatchConfig(rc)))
	defer server.Close()

	override := "Temporary override."
	body, _ := json.Marshal(ConfigPatch{Judge: &JudgeConfigPatch{Prompt: &override}})
	resp, err := http.DefaultClient.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	resp.Body.Close()

	empty := ""
	body, _ = json.Marshal(ConfigPatch{Judge: &JudgeConfigPatch{Prompt: &empty}})
	resp, err = http.DefaultClient.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var view ConfigView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Judge.PromptOverride != "" {
		t.Errorf("PromptOverride = %q, want empty", view.Judge.PromptOverride)
	}
	if !strings.Contains(view.Judge.EffectivePrompt, "You are a League of Legends tactical assistant") {
		t.Errorf("default prompt not restored")
	}
}

func TestHandlePatchConfig_LanguageAndPromptTogether(t *testing.T) {
	rc := setupRuntimeConfig()
	server := httptest.NewServer(http.HandlerFunc(handlePatchConfig(rc)))
	defer server.Close()

	lang := "pt-BR"
	override := "Foque em objetivos."
	body, _ := json.Marshal(ConfigPatch{Judge: &JudgeConfigPatch{Language: &lang, Prompt: &override}})
	resp, err := http.DefaultClient.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("patch request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var view ConfigView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Judge.Language != "pt-BR" {
		t.Errorf("Language = %q, want pt-BR", view.Judge.Language)
	}
	if view.Judge.PromptOverride != override {
		t.Errorf("PromptOverride = %q, want %q", view.Judge.PromptOverride, override)
	}
	if !strings.Contains(view.Judge.EffectivePrompt, "Brazilian Portuguese") {
		t.Errorf("EffectivePrompt missing updated language directive")
	}
}
