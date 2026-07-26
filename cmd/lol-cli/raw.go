package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"

	"lol-telemetry/pkg/service"
)

// runRawClient connects to the daemon and prints every WebSocket message.
// It is useful for headless environments or quick debugging.
func runRawClient(daemonAddr string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, daemonAddr, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", daemonAddr, err)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\n", daemonAddr)
	for {
		select {
		case <-ctx.Done():
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			fmt.Println("\nDisconnected")
			return nil
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				fmt.Println("\nDaemon closed the connection")
				return nil
			}
			return err
		}
		printRawMessage(msg)
	}
}

func printRawMessage(msg []byte) {
	var envelope service.WSMessage
	if err := json.Unmarshal(msg, &envelope); err != nil {
		fmt.Printf("[parse error] %v\n", err)
		return
	}

	switch envelope.Type {
	case service.MsgTypeGameState:
		var gs service.GameState
		json.Unmarshal(envelope.Payload, &gs)
		fmt.Printf("[game] %s time=%.1f players=%d\n", gs.GameMode, gs.GameTime, len(gs.Players))
	case service.MsgTypeJudgeAdvice:
		var advice service.JudgeAdvice
		json.Unmarshal(envelope.Payload, &advice)
		fmt.Printf("[judge] %s\n", advice.Advice)
	case service.MsgTypeEvent:
		var event service.EventMessage
		json.Unmarshal(envelope.Payload, &event)
		fmt.Printf("[event] %s at %.1f\n", event.EventName, event.EventTime)
	case service.MsgTypeError:
		var errMsg service.ErrorMessage
		json.Unmarshal(envelope.Payload, &errMsg)
		fmt.Printf("[error] %s: %s\n", errMsg.Code, errMsg.Message)
	default:
		fmt.Printf("[%s] %s\n", envelope.Type, string(envelope.Payload))
	}
}
