package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"

	"lol-telemetry/internal/tui"
	"lol-telemetry/pkg/service"
)

func main() {
	defer service.LogPanic()

	cleanup, err := service.SetupLogging("lol-cli", "LOL_CLI_LOG")
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging setup failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	var daemonAddr string
	var rawMode bool
	var noAlt bool
	var check bool
	flag.StringVar(&daemonAddr, "daemon", "ws://localhost:8080/ws", "daemon WebSocket address")
	flag.BoolVar(&rawMode, "raw", false, "print raw WebSocket messages instead of launching the TUI")
	flag.BoolVar(&noAlt, "no-alt", false, "disable alternate screen buffer (useful for some Windows terminals)")
	flag.BoolVar(&check, "check", false, "test the daemon WebSocket connection and exit")
	flag.Parse()

	log.Printf("flags: daemon=%s raw=%v noAlt=%v check=%v", daemonAddr, rawMode, noAlt, check)

	if check {
		if err := checkDaemon(daemonAddr); err != nil {
			service.Fatal("daemon check failed: %v", err)
		}
		fmt.Printf("daemon reachable at %s\n", daemonAddr)
		return
	}

	if rawMode || !isTTY() {
		if !rawMode && !isTTY() {
			log.Printf("stdin is not a TTY; falling back to raw mode")
		}
		if err := runRawClient(daemonAddr); err != nil {
			service.Fatal("raw client error: %v", err)
		}
		return
	}

	opts := []tea.ProgramOption{}
	if !noAlt {
		opts = append(opts, tea.WithAltScreen())
	}
	program := tea.NewProgram(tui.NewModel(daemonAddr), opts...)
	if _, err := program.Run(); err != nil {
		service.Fatal("TUI error: %v", err)
	}
}

func isTTY() bool {
	return term.IsTerminal(os.Stdin.Fd())
}
