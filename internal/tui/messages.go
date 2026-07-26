// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"lol-telemetry/pkg/service"
)

// ConnectedMsg is sent when the WebSocket connection is established.
type ConnectedMsg struct{}

// DisconnectedMsg is sent when the WebSocket connection is lost.
type DisconnectedMsg struct{ Error error }

// GameStateMsg carries a snapshot from the daemon.
type GameStateMsg service.GameState

// JudgeAdviceMsg carries advice from any hook.
type JudgeAdviceMsg service.JudgeAdvice

// EventMsg carries a game event from the daemon.
type EventMsg service.EventMessage

// RawMsg carries a raw WS envelope for the log view.
type RawMsg service.WSMessage
