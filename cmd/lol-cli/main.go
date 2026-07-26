// Command lol-cli is the interactive TUI client for the lol-telemetry daemon.
// It connects to the daemon over WebSocket and displays live game state, Judge
// advice, game events, and raw messages. The daemon must be running first:
//
//	go run ./cmd/lol-daemon
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"

	"lol-telemetry/internal/tui"
)

func main() {
	defer logPanic()

	cleanup, err := setupLogging()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging setup failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	var daemonAddr string
	var rawMode bool
	var noAlt bool
	flag.StringVar(&daemonAddr, "daemon", "ws://localhost:8080/ws", "daemon WebSocket address")
	flag.BoolVar(&rawMode, "raw", false, "print raw WebSocket messages instead of launching the TUI")
	flag.BoolVar(&noAlt, "no-alt", false, "disable alternate screen buffer (useful for some Windows terminals)")
	flag.Parse()

	log.Printf("flags: daemon=%s raw=%v noAlt=%v", daemonAddr, rawMode, noAlt)

	if rawMode || !isTTY() {
		if !rawMode && !isTTY() {
			log.Printf("stdin is not a TTY; falling back to raw mode")
		}
		if err := runRawClient(daemonAddr); err != nil {
			fatal("raw client error: %v", err)
		}
		return
	}

	opts := []tea.ProgramOption{}
	if !noAlt {
		opts = append(opts, tea.WithAltScreen())
	}
	program := tea.NewProgram(tui.NewModel(daemonAddr), opts...)
	if _, err := program.Run(); err != nil {
		fatal("TUI error: %v", err)
	}
}

// isTTY reports whether stdin is an interactive terminal.
func isTTY() bool {
	return term.IsTerminal(os.Stdin.Fd())
}
