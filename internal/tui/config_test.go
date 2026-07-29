package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/pkg/service"
)

func TestEditPromptCmd_SavesChangedPrompt(t *testing.T) {
	t.Setenv("EDITOR", "cat")

	cmd, err := editPromptCmd("Original prompt.")
	if err != nil {
		t.Fatalf("editPromptCmd: %v", err)
	}

	msg := cmd().(PromptEditorResultMsg)
	if msg.Err != nil {
		t.Fatalf("editor result error: %v", msg.Err)
	}
	if strings.TrimSpace(msg.Content) != "Original prompt." {
		t.Errorf("content = %q, want Original prompt.", msg.Content)
	}
}

func TestEditPromptCmd_NoEditor(t *testing.T) {
	t.Setenv("EDITOR", "")
	_, err := editPromptCmd("prompt")
	if err == nil {
		t.Fatal("expected error when EDITOR is unset")
	}
}

func TestEditPromptCmd_FakeEditorChangesPrompt(t *testing.T) {
	t.Setenv("EDITOR", "true")

	// Restore real runner after test.
	orig := editorRunner
	defer func() { editorRunner = orig }()

	editorRunner = func(path string) tea.Cmd {
		return func() tea.Msg {
			if err := os.WriteFile(path, []byte("Changed prompt."), 0o644); err != nil {
				return err
			}
			return nil
		}
	}

	cmd, err := editPromptCmd("Original prompt.")
	if err != nil {
		t.Fatalf("editPromptCmd: %v", err)
	}

	msg := cmd().(PromptEditorResultMsg)
	if msg.Err != nil {
		t.Fatalf("editor result error: %v", msg.Err)
	}
	if strings.TrimSpace(msg.Content) != "Changed prompt." {
		t.Errorf("content = %q, want Changed prompt.", msg.Content)
	}
}

func TestPromptOverrideLabel(t *testing.T) {
	if got := promptOverrideLabel(""); got != "(default)" {
		t.Errorf("promptOverrideLabel(\"\") = %q, want (default)", got)
	}
	if got := promptOverrideLabel("short"); got != "short" {
		t.Errorf("promptOverrideLabel(\"short\") = %q, want short", got)
	}
	long := strings.Repeat("a", 50)
	if got := promptOverrideLabel(long); !strings.HasSuffix(got, "...") {
		t.Errorf("promptOverrideLabel(long) = %q, want truncated", got)
	}
}

func TestConfigState_ViewShowsEffectivePrompt(t *testing.T) {
	s := newConfigState()
	s.Update(service.ConfigView{
		Judge: service.JudgeConfigView{
			Language:        "en",
			PromptOverride:  "Custom.",
			EffectivePrompt: "Custom. Respond in English.",
		},
		Hooks: []hooks.HookView{},
	})
	view := s.View(80, 24)
	if !strings.Contains(view, "Custom.") {
		t.Errorf("view missing override label, got:\n%s", view)
	}
	if !strings.Contains(view, "Respond in English") {
		t.Errorf("view missing effective prompt, got:\n%s", view)
	}
}
