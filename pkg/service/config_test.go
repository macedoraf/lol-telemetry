package service

import (
	"os"
	"testing"
	"time"
)

func TestLoadDaemonConfigFromEnv_Defaults(t *testing.T) {
	cfg := LoadDaemonConfigFromEnv()
	if cfg.Port != "8080" {
		t.Errorf("default port = %q, want 8080", cfg.Port)
	}
	if cfg.PollInterval != 1*time.Second {
		t.Errorf("default poll interval = %v, want 1s", cfg.PollInterval)
	}
	if cfg.JudgeLanguage != "en" {
		t.Errorf("default language = %q, want en", cfg.JudgeLanguage)
	}
	if cfg.RecordEnabled {
		t.Errorf("default RecordEnabled = %v, want false", cfg.RecordEnabled)
	}
	if cfg.RecordingsDir != "./recordings" {
		t.Errorf("default RecordingsDir = %q, want ./recordings", cfg.RecordingsDir)
	}
}

func TestLoadDaemonConfigFromEnv_Recording(t *testing.T) {
	t.Setenv("LOL_RECORD_ENABLED", "true")
	t.Setenv("LOL_RECORDINGS_DIR", "/tmp/lol-recs")
	cfg := LoadDaemonConfigFromEnv()
	if !cfg.RecordEnabled {
		t.Errorf("RecordEnabled = %v, want true", cfg.RecordEnabled)
	}
	if cfg.RecordingsDir != "/tmp/lol-recs" {
		t.Errorf("RecordingsDir = %q, want /tmp/lol-recs", cfg.RecordingsDir)
	}
}

func TestLoadDaemonConfigFromEnv_JudgeLanguage(t *testing.T) {
	tests := []struct {
		envVal string
		want   string
	}{
		{"", "en"},
		{"pt-BR", "pt-BR"},
		{"es", "es"},
		{"fr", "fr"},
	}
	for _, tc := range tests {
		t.Run(tc.envVal, func(t *testing.T) {
			if tc.envVal != "" {
				os.Setenv("JUDGE_LANGUAGE", tc.envVal)
				defer os.Unsetenv("JUDGE_LANGUAGE")
			} else {
				os.Unsetenv("JUDGE_LANGUAGE")
			}
			cfg := LoadDaemonConfigFromEnv()
			if cfg.JudgeLanguage != tc.want {
				t.Errorf("JudgeLanguage = %q, want %q", cfg.JudgeLanguage, tc.want)
			}
		})
	}
}
