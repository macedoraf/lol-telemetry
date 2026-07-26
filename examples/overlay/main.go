// Command overlay demonstrates how to integrate an external tool with the
// lol-telemetry daemon over WebSocket. The daemon polls the LoL Live Client
// Data API and pushes game state, events, and judge advice to the overlay.
//
// Before running this example, start the daemon:
//
//	go run ./cmd/lol-daemon
//
// Then run this example:
//
//	go run ./examples/overlay
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"

	"lol-telemetry/pkg/service"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", "ws://localhost:8080/ws", "lol-telemetry daemon WebSocket address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, addr, nil)
	if err != nil {
		log.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\n", addr)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down overlay...")
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Println("Connection closed by server")
				return
			}
			log.Printf("read error: %v", err)
			return
		}

		handleMessage(msg)
	}
}

func handleMessage(msg []byte) {
	var envelope service.WSMessage
	if err := json.Unmarshal(msg, &envelope); err != nil {
		log.Printf("unmarshal: %v", err)
		return
	}

	switch envelope.Type {
	case service.MsgTypeHello:
		fmt.Println("[hello] daemon ready")

	case service.MsgTypeGameState:
		var gs service.GameState
		if err := json.Unmarshal(envelope.Payload, &gs); err != nil {
			log.Printf("game state unmarshal: %v", err)
			return
		}
		fmt.Printf("[%s] gameTime=%.1f players=%d\n", gs.GameMode, gs.GameTime, len(gs.Players))

	case service.MsgTypeJudgeAdvice:
		var advice service.JudgeAdvice
		if err := json.Unmarshal(envelope.Payload, &advice); err != nil {
			log.Printf("advice unmarshal: %v", err)
			return
		}
		fmt.Printf("[judge %s@min%d] %s\n", advice.HookName, advice.GameMinute, advice.Advice)

	case service.MsgTypeEvent:
		var event service.EventMessage
		if err := json.Unmarshal(envelope.Payload, &event); err != nil {
			log.Printf("event unmarshal: %v", err)
			return
		}
		fmt.Printf("[event] %s at %.1f\n", event.EventName, event.EventTime)

	case service.MsgTypeError:
		var errMsg service.ErrorMessage
		if err := json.Unmarshal(envelope.Payload, &errMsg); err != nil {
			log.Printf("error unmarshal: %v", err)
			return
		}
		fmt.Printf("[error] %s: %s\n", errMsg.Code, errMsg.Message)

	default:
		fmt.Printf("[unknown] type=%s\n", envelope.Type)
	}
}
