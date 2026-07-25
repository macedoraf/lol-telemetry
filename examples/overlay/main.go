// Command overlay demonstrates how to poll the Live Client Data API to build a
// simple game overlay. It is a self-contained example referenced from the
// README; it is not intended to be run during a real League of Legends match
// unless the LoL client is running and serving the local API on port 2999.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lol-telemetry/pkg/riotclient"
)

func main() {
	// The SDK client is preconfigured with InsecureSkipVerify so it can talk
	// to the self-signed certificate served by the LoL client on 127.0.0.1:2999.
	client := riotclient.NewClient()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Fetch the active player name once at startup.
	name, err := client.GetActivePlayerName()
	if err != nil {
		log.Printf("Unable to fetch active player name (is a game running?): %v", err)
	} else {
		fmt.Printf("Active player: %s\n", name)
	}

	// Poll the local server at a conservative rate. Riot recommends keeping
	// Live Client Data API traffic low to avoid impacting the LoL client.
	// 1 request per second is a reasonable maximum for a lightweight overlay.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down overlay...")
			return
		case <-ticker.C:
			fetchOverlayData(client)
		}
	}
}

func fetchOverlayData(client *riotclient.Client) {
	// Game stats: game mode, current game time, map information.
	stats, err := client.GetGameStats()
	if err != nil {
		log.Printf("game stats error: %v", err)
		return
	}
	fmt.Printf("[%s] Game time: %.2f\n", stats.GameMode, stats.GameTime)

	// Event data: recent in-game events such as GameStart, ChampionKill, etc.
	events, err := client.GetEventData()
	if err != nil {
		log.Printf("event data error: %v", err)
		return
	}
	if len(events.Events) > 0 {
		last := events.Events[len(events.Events)-1]
		fmt.Printf("Last event: %s (t=%.2f)\n", last.EventName, last.EventTime)
	}

	// In a real overlay, the data above would be pushed to a UI layer (Wails,
	// Electron, a web socket, etc.) instead of being printed to stdout.
}
