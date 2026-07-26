// Package tui implements the Bubble Tea interface for the lol-telemetry daemon client.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/pkg/service"
)

const dialTimeout = 5 * time.Second

// WSClient connects to the daemon and translates WebSocket messages into tea.Msg.
type WSClient struct {
	addr string
	conn *websocket.Conn
}

// NewWSClient creates a client for the given WebSocket address.
func NewWSClient(addr string) *WSClient {
	return &WSClient{addr: addr}
}

// Connect dials the daemon and returns a command that reads messages.
func (c *WSClient) Connect(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), dialTimeout)
			defer cancel()
		}
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.addr, nil)
		if err != nil {
			return DisconnectedMsg{Error: fmt.Errorf("dial %s: %w", c.addr, err)}
		}
		c.conn = conn
		return ConnectedMsg{}
	}
}

// Read returns a command that reads one message from the daemon.
func (c *WSClient) Read() tea.Cmd {
	return func() tea.Msg {
		if c.conn == nil {
			return DisconnectedMsg{Error: fmt.Errorf("not connected")}
		}
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return DisconnectedMsg{Error: err}
		}
		return parseMessage(data)
	}
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func parseMessage(data []byte) tea.Msg {
	var envelope service.WSMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return RawMsg{Type: "parse_error", Payload: data}
	}

	switch envelope.Type {
	case service.MsgTypeGameState:
		var gs service.GameState
		if err := json.Unmarshal(envelope.Payload, &gs); err != nil {
			return RawMsg(envelope)
		}
		return GameStateMsg(gs)

	case service.MsgTypeJudgeAdvice:
		var advice service.JudgeAdvice
		if err := json.Unmarshal(envelope.Payload, &advice); err != nil {
			return RawMsg(envelope)
		}
		return JudgeAdviceMsg(advice)

	case service.MsgTypeEvent:
		var event service.EventMessage
		if err := json.Unmarshal(envelope.Payload, &event); err != nil {
			return RawMsg(envelope)
		}
		return EventMsg(event)

	default:
		return RawMsg(envelope)
	}
}
