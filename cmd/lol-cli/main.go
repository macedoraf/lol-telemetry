// Command lol-cli is the interactive TUI client for the lol-telemetry daemon.
// It connects to the daemon over WebSocket and displays live game state, Judge
// advice, game events, and raw messages. The daemon must be running first:
//
//	go run ./cmd/lol-daemon
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lol-telemetry/internal/tui"
)

func main() {
	var daemonAddr string
	var rawMode bool
	flag.StringVar(&daemonAddr, "daemon", "ws://localhost:8080/ws", "daemon WebSocket address")
	flag.BoolVar(&rawMode, "raw", false, "print raw WebSocket messages instead of launching the TUI")
	flag.Parse()

	if rawMode {
		if err := runRawClient(daemonAddr); err != nil {
			fmt.Fprintf(os.Stderr, "raw client error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	program := tea.NewProgram(tui.NewModel(daemonAddr), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
