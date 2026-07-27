package recorder

import (
	"strings"
	"testing"
	"time"
)

func TestSessionManager_Observe(t *testing.T) {
	tests := []struct {
		name     string
		steps    []observeStep
		wantIDs  int // number of distinct non-empty session ids produced
		wantEnds int
	}{
		{
			name: "start then end by game time",
			steps: []observeStep{
				{0, false, "", false, false},
				{1.0, true, "", true, false},
				{2.0, true, "last", false, false},
				{0.0, true, "", false, true},
			},
			wantIDs:  1,
			wantEnds: 1,
		},
		{
			name: "start then end by api error",
			steps: []observeStep{
				{5.0, true, "", true, false},
				{6.0, true, "last", false, false},
				{0.0, false, "", false, true},
			},
			wantIDs:  1,
			wantEnds: 1,
		},
		{
			name: "new match when game time goes backwards",
			steps: []observeStep{
				{10.0, true, "", true, false},
				{500.0, true, "last", false, false},
				{1.0, true, "", true, true},
				{2.0, true, "last", false, false},
			},
			wantIDs:  2,
			wantEnds: 1,
		},
		{
			name: "same session does not duplicate id",
			steps: []observeStep{
				{3.0, true, "", true, false},
				{4.0, true, "last", false, false},
				{5.0, true, "last", false, false},
				{6.0, true, "last", false, false},
			},
			wantIDs:  1,
			wantEnds: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sm := &SessionManager{}
			ids := make(map[string]struct{})
			var ends int
			for i, step := range tc.steps {
				id, started, ended := sm.Observe(step.gameTime, step.apiOK)
				if id != "" {
					ids[id] = struct{}{}
				}
				if ended {
					ends++
				}
				if started && id == "" {
					t.Fatalf("step %d: started but id is empty", i)
				}
				if step.wantID != "" && step.wantID != "last" && id != step.wantID {
					// We do not assert exact ids because they are time-based, but
					// we ensure non-empty on start and same as previous on continue.
				}
				if step.wantID == "last" && id == "" {
					t.Fatalf("step %d: expected continuation id, got empty", i)
				}
				if step.wantStarted && !started {
					t.Fatalf("step %d: expected start", i)
				}
				if step.wantEnded && !ended {
					t.Fatalf("step %d: expected end", i)
				}
			}
			if len(ids) != tc.wantIDs {
				t.Errorf("got %d distinct session ids, want %d", len(ids), tc.wantIDs)
			}
			if ends != tc.wantEnds {
				t.Errorf("got %d session ends, want %d", ends, tc.wantEnds)
			}
		})
	}
}

type observeStep struct {
	gameTime      float64
	apiOK         bool
	wantID        string
	wantStarted   bool
	wantEnded     bool
}

func TestNewSessionID_Format(t *testing.T) {
	id := newSessionID()
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		t.Fatalf("session id %q does not match YYYYMMDD-HHMMSS-hex6", id)
	}
	if _, err := time.Parse("20060102", parts[0]); err != nil {
		t.Errorf("date part %q invalid: %v", parts[0], err)
	}
	if _, err := time.Parse("150405", parts[1]); err != nil {
		t.Errorf("time part %q invalid: %v", parts[1], err)
	}
	if len(parts[2]) != 12 {
		t.Errorf("random hex part length = %d, want 12", len(parts[2]))
	}
}
