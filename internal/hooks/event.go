// Package hooks contains trigger conditions that decide when to invoke the Judge.
package hooks

import (
	"fmt"

	"lol-telemetry/internal/types"
	"lol-telemetry/pkg/riotclient"
)

// Event hook names.
const (
	GameStartHookName      = "game-start"
	DeathHookName          = "player-death"
	RecallHookName         = "recall"
	AllyGoldSpikeHookName  = "ally-gold-spike"
	EnemyGoldSpikeHookName = "enemy-gold-spike"
	FirstTurretHookName    = "first-turret"
	LaningPhaseEndHookName = "laning-phase-end"
)

// GameStartHook fires once when the match transitions from no active game time to positive game time.
type GameStartHook struct{}

func (GameStartHook) Name() string { return GameStartHookName }
func (GameStartHook) Instruction() string {
	return "Give one early-game advice: lane setup, level 1 positioning, or warding based on the matchup."
}
func (GameStartHook) CurrentMark(float64) int64 { return 0 }
func (GameStartHook) ShouldFire(ctx types.HookContext) (bool, error) {
	return ctx.GameTime > 0 && ctx.PrevData.GameData.GameTime <= 0, nil
}

// PlayerDeathHook fires when the active player becomes dead.
type PlayerDeathHook struct{}

func (PlayerDeathHook) Name() string { return DeathHookName }
func (PlayerDeathHook) Instruction() string {
	return "I just died. What should I watch or do on my next respawn: build path, map state, or objective timer?"
}
func (PlayerDeathHook) CurrentMark(float64) int64 { return 0 }
func (PlayerDeathHook) ShouldFire(ctx types.HookContext) (bool, error) {
	active, ok := riotclient.FindActivePlayer(ctx.Data)
	if !ok {
		return false, fmt.Errorf("active player not found")
	}
	prevActive, _ := riotclient.FindActivePlayer(ctx.PrevData)
	return active.IsDead && (!prevActive.IsDead || prevActive.SummonerName == ""), nil
}

// RecallHook fires when the active player has accumulated enough unspent gold for a key item.
type RecallHook struct {
	GoldThreshold      float64
	MinGameTimeSeconds float64
}

func (h *RecallHook) Name() string { return RecallHookName }
func (h *RecallHook) Instruction() string {
	return "I have a lot of unspent gold. Should I recall now, or keep farming/pressuring?"
}
func (h *RecallHook) CurrentMark(float64) int64 { return 0 }
func (h *RecallHook) goldThreshold() float64 {
	if h.GoldThreshold > 0 {
		return h.GoldThreshold
	}
	return 1000
}
func (h *RecallHook) minGameTime() float64 {
	if h.MinGameTimeSeconds > 0 {
		return h.MinGameTimeSeconds
	}
	return 60
}
func (h *RecallHook) ShouldFire(ctx types.HookContext) (bool, error) {
	active, ok := riotclient.FindActivePlayer(ctx.Data)
	if !ok {
		return false, fmt.Errorf("active player not found")
	}
	if active.IsDead {
		return false, nil
	}
	prevActive, hadPrev := riotclient.FindActivePlayer(ctx.PrevData)
	threshold := h.goldThreshold()
	gold := ctx.Data.ActivePlayer.CurrentGold
	if gold < threshold {
		return false, nil
	}
	if !hadPrev || prevActive.SummonerName == "" {
		return false, nil
	}
	if ctx.Data.GameData.GameTime < h.minGameTime() {
		return false, nil
	}
	return ctx.PrevData.ActivePlayer.CurrentGold < threshold, nil
}

// Configure implements Configurable.
func (h *RecallHook) Configure(params map[string]any) error {
	if v, ok := params["goldThreshold"]; ok {
		f, ok2 := toFloat(v)
		if !ok2 || f < 0 {
			return fmt.Errorf("goldThreshold must be a number >= 0, got %v", v)
		}
		h.GoldThreshold = f
	}
	if v, ok := params["minGameTimeSeconds"]; ok {
		f, ok2 := toFloat(v)
		if !ok2 || f < 0 {
			return fmt.Errorf("minGameTimeSeconds must be a number >= 0, got %v", v)
		}
		h.MinGameTimeSeconds = f
	}
	return nil
}

// Spec implements Configurable.
func (h *RecallHook) Spec() map[string]ParamSpec {
	return map[string]ParamSpec{
		"goldThreshold":      {Type: "float", Default: 1000, Min: 0},
		"minGameTimeSeconds": {Type: "float", Default: 60, Min: 0},
	}
}

// AllyGoldSpikeHook fires when the active player buys a significant item, indicating a power spike.
type AllyGoldSpikeHook struct{}

func (AllyGoldSpikeHook) Name() string { return AllyGoldSpikeHookName }
func (AllyGoldSpikeHook) Instruction() string {
	return "I just completed a major item or power spike. How should I exploit it in the next minute?"
}
func (AllyGoldSpikeHook) CurrentMark(float64) int64 { return 0 }
func (AllyGoldSpikeHook) ShouldFire(ctx types.HookContext) (bool, error) {
	active, ok := riotclient.FindActivePlayer(ctx.Data)
	if !ok {
		return false, fmt.Errorf("active player not found")
	}
	prevActive, hadPrev := riotclient.FindActivePlayer(ctx.PrevData)
	if !hadPrev || prevActive.SummonerName == "" {
		return false, nil
	}
	return riotclient.ItemCount(active.Items) > riotclient.ItemCount(prevActive.Items), nil
}

// EnemyGoldSpikeHook fires when the lane opponent buys a new item or levels up significantly.
type EnemyGoldSpikeHook struct{}

func (EnemyGoldSpikeHook) Name() string { return EnemyGoldSpikeHookName }
func (EnemyGoldSpikeHook) Instruction() string {
	return "My lane opponent just got a new item or level advantage. How should I respect or punish it?"
}
func (EnemyGoldSpikeHook) CurrentMark(float64) int64 { return 0 }
func (EnemyGoldSpikeHook) ShouldFire(ctx types.HookContext) (bool, error) {
	active, ok := riotclient.FindActivePlayer(ctx.Data)
	if !ok {
		return false, fmt.Errorf("active player not found")
	}
	opponent := riotclient.FindOpponent(ctx.Data, active.Position, active.Team)
	if opponent.SummonerName == "" {
		return false, nil
	}
	prevOpponent := riotclient.FindOpponent(ctx.PrevData, active.Position, active.Team)
	if prevOpponent.SummonerName == "" {
		return false, nil
	}
	return riotclient.ItemCount(opponent.Items) > riotclient.ItemCount(prevOpponent.Items) || opponent.Level > prevOpponent.Level, nil
}

// FirstTurretHook fires when the first outer turret is destroyed.
type FirstTurretHook struct{}

func (FirstTurretHook) Name() string { return FirstTurretHookName }
func (FirstTurretHook) Instruction() string {
	return "First turret fell. How does the map open up and where should I rotate next?"
}
func (FirstTurretHook) CurrentMark(float64) int64 { return 0 }
func (FirstTurretHook) ShouldFire(ctx types.HookContext) (bool, error) {
	return firstTurretEventOccurred(ctx.Data.Events.Events) && !firstTurretEventOccurred(ctx.PrevData.Events.Events), nil
}

// LaningPhaseEndHook fires once when the game reaches the configured mark (default 14min).
type LaningPhaseEndHook struct {
	MarkSeconds float64
}

func (h *LaningPhaseEndHook) Name() string { return LaningPhaseEndHookName }
func (h *LaningPhaseEndHook) Instruction() string {
	return "Laning phase is ending. Give one macro priority: rotate, group for objective, or secure side lane farm."
}
func (h *LaningPhaseEndHook) CurrentMark(float64) int64 { return 0 }
func (h *LaningPhaseEndHook) mark() float64 {
	if h.MarkSeconds > 0 {
		return h.MarkSeconds
	}
	return 840
}
func (h *LaningPhaseEndHook) ShouldFire(ctx types.HookContext) (bool, error) {
	m := h.mark()
	return ctx.GameTime >= m && ctx.PrevData.GameData.GameTime < m, nil
}

// Configure implements Configurable.
func (h *LaningPhaseEndHook) Configure(params map[string]any) error {
	if v, ok := params["markSeconds"]; ok {
		f, ok2 := toFloat(v)
		if !ok2 || f < 0 {
			return fmt.Errorf("markSeconds must be a number >= 0, got %v", v)
		}
		h.MarkSeconds = f
	}
	return nil
}

// Spec implements Configurable.
func (h *LaningPhaseEndHook) Spec() map[string]ParamSpec {
	return map[string]ParamSpec{
		"markSeconds": {Type: "float", Default: 840, Min: 0},
	}
}

func firstTurretEventOccurred(events []riotclient.Event) bool {
	for _, e := range events {
		if e.EventName == "TurretKilled" || e.EventName == "FirstBrick" {
			return true
		}
	}
	return false
}
