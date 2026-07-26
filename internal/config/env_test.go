package config

import (
	"strings"
	"testing"
)

func TestEnvConfig_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		cfg     EnvConfig
		enabled bool
	}{
		{"with key", EnvConfig{APIKey: "sk-123", Model: defaultModel}, true},
		{"without key", EnvConfig{APIKey: "", Model: defaultModel}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.Enabled(); got != tt.enabled {
				t.Errorf("Enabled() = %v, want %v", got, tt.enabled)
			}
		})
	}
}

func TestEnvConfig_MaskedKey(t *testing.T) {
	tests := []struct {
		name string
		cfg  EnvConfig
		want string
	}{
		{"empty", EnvConfig{APIKey: ""}, "(não configurada)"},
		{"short", EnvConfig{APIKey: "abc"}, "****"},
		{"normal", EnvConfig{APIKey: "sk-1234567890abcdef"}, "...cdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.MaskedKey(); got != tt.want {
				t.Errorf("MaskedKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnvConfig_String(t *testing.T) {
	cfg := EnvConfig{APIKey: "sk-secret", Model: "openai/gpt-4o-mini"}
	s := cfg.String()
	if !strings.Contains(s, "ativado") {
		t.Errorf("String() missing enabled status, got %q", s)
	}
	if !strings.Contains(s, "openai/gpt-4o-mini") {
		t.Errorf("String() missing model, got %q", s)
	}
	if strings.Contains(s, "sk-secret") {
		t.Errorf("String() should not expose raw key, got %q", s)
	}
}
