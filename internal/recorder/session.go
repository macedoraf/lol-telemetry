package recorder

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// SessionManager detects game-session boundaries from the gameTime stream.
// It is not safe for concurrent use; the caller serializes Observe calls.
type SessionManager struct {
	active      bool
	sessionID   string
	lastGameTime float64
}

// Observe consumes one observation and returns the current session id plus
// whether a session started or ended this tick.
//
// State machine:
//   - inativo → ativo (gameTime > 0 and apiOK): new session.
//   - ativo → inativo (!apiOK or gameTime <= 0): session ended.
//   - gameTime went backwards while active: previous session ended and a new one started.
func (s *SessionManager) Observe(gameTime float64, apiOK bool) (id string, started bool, ended bool) {
	active := apiOK && gameTime > 0

	if !s.active && active {
		s.sessionID = newSessionID()
		s.active = true
		s.lastGameTime = gameTime
		return s.sessionID, true, false
	}

	if s.active && !active {
		id = s.sessionID
		s.active = false
		s.sessionID = ""
		s.lastGameTime = 0
		return id, false, true
	}

	if s.active && active && gameTime < s.lastGameTime {
		// Game time went backwards => new match started.
		id = s.sessionID
		newID := newSessionID()
		s.sessionID = newID
		s.lastGameTime = gameTime
		return newID, true, true
	}

	if s.active {
		s.lastGameTime = gameTime
	}
	return s.sessionID, false, false
}

// Active reports whether a session is currently open.
func (s *SessionManager) Active() bool {
	return s.active
}

// SessionID returns the current session id, or "" if none is active.
func (s *SessionManager) SessionID() string {
	return s.sessionID
}

func newSessionID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// Fallback should be unreachable in practice; use timestamp suffix.
		return fmt.Sprintf("%s-%06x", time.Now().UTC().Format("20060102-150405"), time.Now().UnixNano()%0xffffff)
	}
	return fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102-150405"), hex.EncodeToString(b))
}
