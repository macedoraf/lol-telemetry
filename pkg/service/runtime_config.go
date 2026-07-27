// Package service provides the background daemon and WebSocket API for lol-telemetry.
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/orchestrator"
)

// RuntimeConfig is a thread-safe holder for the live runtime configuration.
// It buffers the last successfully applied config for GET and serializes mutations.
type RuntimeConfig struct {
	mu      sync.RWMutex
	lang    string // normalized language code
	manager *runtimeManager
}

type runtimeManager struct {
	reg     *hooks.Registry
	builder *payload.Builder
	orch    *orchestrator.Orchestrator
}

// NewRuntimeConfig creates a runtime config from initial boot values.
func NewRuntimeConfig(initialLang string, reg *hooks.Registry, builder *payload.Builder, orch *orchestrator.Orchestrator) *RuntimeConfig {
	rc := &RuntimeConfig{
		lang: initialLang,
		manager: &runtimeManager{
			reg:     reg,
			builder: builder,
			orch:    orch,
		},
	}
	return rc
}

// Snapshot returns the current config view.
func (rc *RuntimeConfig) Snapshot() ConfigView {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return ConfigView{
		Judge: JudgeConfigView{
			Language:        rc.lang,
			PromptOverride:  rc.manager.builder.PromptOverride(),
			EffectivePrompt: rc.manager.builder.EffectivePrompt(),
		},
		Hooks: rc.manager.reg.Snapshot(),
	}
}

// Apply applies a partial config patch and validates it against the live system.
func (rc *RuntimeConfig) Apply(patch ConfigPatch) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Validate and apply judge section.
	if patch.Judge != nil {
		if patch.Judge.Language != nil {
			norm := payload.NormalizeLanguage(*patch.Judge.Language)
			if norm != *patch.Judge.Language {
				return fmt.Errorf("invalid language %q, valid values: en, pt-BR, es", *patch.Judge.Language)
			}
			rc.manager.builder.SetLanguage(norm)
			rc.lang = norm
		}
		if patch.Judge.Prompt != nil {
			// Validate through the builder first; it will reject invalid prompts.
			if err := rc.manager.builder.SetPromptOverride(*patch.Judge.Prompt); err != nil {
				return err
			}
		}
	}

	// Validate and apply hooks section.
	if len(patch.Hooks) > 0 {
		for _, hp := range patch.Hooks {
			if hp.Enabled != nil {
				if err := rc.manager.reg.SetEnabled(hp.Name, *hp.Enabled); err != nil {
					return err
				}
				rc.manager.orch.ResetHook(hp.Name)
			}
			if hp.Params != nil {
				if err := rc.manager.reg.Configure(hp.Name, hp.Params); err != nil {
					return err
				}
				rc.manager.orch.ResetHook(hp.Name)
			}
		}
	}

	return nil
}

func handleGetConfig(rc *RuntimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rc.Snapshot())
	}
}

func handlePatchConfig(rc *RuntimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var patch ConfigPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"invalid json: %s"}`, err), http.StatusBadRequest)
			return
		}
		if patch.Judge == nil && len(patch.Hooks) == 0 {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}
		if err := rc.Apply(patch); err != nil {
			log.Printf("config patch rejected: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rc.Snapshot())
	}
}
