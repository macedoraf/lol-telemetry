package hooks

import (
	"testing"

	"lol-telemetry/internal/types"
)

func TestPeriodic5MinHook_ShouldFire(t *testing.T) {
	hook := Periodic5MinHook{}
	tests := []struct {
		name      string
		gameTime  float64
		prevFired map[string]int64
		want      bool
	}{
		{
			name:      "no active match",
			gameTime:  0,
			prevFired: map[string]int64{},
			want:      false,
		},
		{
			name:      "before first mark",
			gameTime:  299,
			prevFired: map[string]int64{},
			want:      false,
		},
		{
			name:      "exactly at five minutes",
			gameTime:  300,
			prevFired: map[string]int64{PeriodicHookName: 0},
			want:      true,
		},
		{
			name:      "already fired at five minutes",
			gameTime:  301,
			prevFired: map[string]int64{PeriodicHookName: 300},
			want:      false,
		},
		{
			name:      "first observation establishes baseline",
			gameTime:  450,
			prevFired: map[string]int64{},
			want:      false,
		},
		{
			name:      "first mark after late start",
			gameTime:  450,
			prevFired: map[string]int64{PeriodicHookName: 300},
			want:      false,
		},
		{
			name:      "next mark after five",
			gameTime:  600,
			prevFired: map[string]int64{PeriodicHookName: 300},
			want:      true,
		},
		{
			name:      "deduplicate same mark",
			gameTime:  610,
			prevFired: map[string]int64{PeriodicHookName: 600},
			want:      false,
		},
		{
			name:      "skip earlier marks after late start",
			gameTime:  610,
			prevFired: map[string]int64{PeriodicHookName: 600},
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.HookContext{
				GameTime:  tt.gameTime,
				PrevFired: tt.prevFired,
			}
			got, err := hook.ShouldFire(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ShouldFire() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistry_Evaluate(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Periodic5MinHook{})

	ctx := types.HookContext{
		GameTime:  600,
		PrevFired: map[string]int64{PeriodicHookName: 300},
	}
	triggers, err := reg.Evaluate(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(triggers))
	}
	if triggers[0].HookName != PeriodicHookName {
		t.Errorf("trigger name = %s, want %s", triggers[0].HookName, PeriodicHookName)
	}
}
