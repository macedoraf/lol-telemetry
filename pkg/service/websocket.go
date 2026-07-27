// Package service provides the background daemon and WebSocket API for external clients.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for local dev
	},
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WS client connected: %s (total: %d)", client.addr, len(h.clients))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("WS client disconnected: %s (total: %d)", client.addr, len(h.clients))
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("WS broadcast channel full, dropping message")
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		close(client.send)
	}
	h.clients = make(map[*Client]bool)
}

// Client is a single WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	addr string
}

// ServeWS upgrades the HTTP connection to WebSocket and registers the client.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		addr: r.RemoteAddr,
	}
	client.hub.register <- client
	if err := hub.sendHello(client); err != nil {
		log.Printf("WS hello error for %s: %v", client.addr, err)
		client.conn.Close()
		return
	}

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WS write error: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastGameState sends the current game state to all clients.
func (h *Hub) BroadcastGameState(gs GameState) error {
	data, err := json.Marshal(WSMessage{
		Type:    MsgTypeGameState,
		Payload: mustMarshal(gs),
		Seq:     h.nextSeq(),
		Ts:      time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	h.Broadcast(data)
	return nil
}

// BroadcastAdvice sends a Judge advice to all clients.
func (h *Hub) BroadcastAdvice(hookName string, gameMinute int, advice, reasoning string) error {
	data, err := json.Marshal(WSMessage{
		Type:    MsgTypeJudgeAdvice,
		Payload: mustMarshal(JudgeAdvice{HookName: hookName, GameMinute: gameMinute, Advice: advice, Reasoning: reasoning, Timestamp: time.Now().UnixMilli()}),
		Seq:     h.nextSeq(),
		Ts:      time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	h.Broadcast(data)
	return nil
}

// BroadcastEvent sends a game event to all clients.
func (h *Hub) BroadcastEvent(event EventMessage) error {
	data, err := json.Marshal(WSMessage{
		Type:    MsgTypeEvent,
		Payload: mustMarshal(event),
		Seq:     h.nextSeq(),
		Ts:      time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	h.Broadcast(data)
	return nil
}

// BroadcastStatus sends a status update.
func (h *Hub) BroadcastStatus(status string, detail string) error {
	data, err := json.Marshal(WSMessage{
		Type:    MsgTypeError,
		Payload: mustMarshal(ErrorMessage{Code: status, Message: detail}),
		Seq:     h.nextSeq(),
		Ts:      time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	h.Broadcast(data)
	return nil
}

// helloMessage returns the serialized hello envelope.
func (h *Hub) helloMessage() ([]byte, error) {
	return json.Marshal(WSMessage{
		Type:    MsgTypeHello,
		Payload: mustMarshal(HelloMessage{Version: "1.0.0", ServerTS: time.Now().UnixMilli(), Protocol: "lol-telemetry/ws/v1"}),
		Seq:     h.nextSeq(),
		Ts:      time.Now().UnixMilli(),
	})
}

// sendHello sends the hello message directly to a single client.
func (h *Hub) sendHello(c *Client) error {
	data, err := h.helloMessage()
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
	default:
		close(c.send)
		return fmt.Errorf("client send channel full")
	}
	return nil
}

// BroadcastHello sends the hello message to all connected clients.
// ponytail: kept for backward compatibility; new clients receive hello on connect.
func (h *Hub) BroadcastHello() error {
	data, err := h.helloMessage()
	if err != nil {
		return err
	}
	h.Broadcast(data)
	return nil
}

var seqCounter int64
var seqMu sync.Mutex

func (h *Hub) nextSeq() int64 {
	seqMu.Lock()
	defer seqMu.Unlock()
	seqCounter++
	return seqCounter
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
