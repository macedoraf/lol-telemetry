// Command lol-daemon runs the lol-telemetry SDK as a background service.
// It polls the League of Legends Live Client Data API and exposes game state,
// events, and judge advice over WebSocket for external tools like overlays.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"lol-telemetry/pkg/service"
)

func main() {
	defer logPanic()

	cleanup, err := setupLogging()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging setup failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	cfg := service.LoadDaemonConfigFromEnv()

	flag.StringVar(&cfg.Port, "port", cfg.Port, "WebSocket server port")
	flag.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "Live Client Data API polling interval")
	flag.Parse()

	log.Printf("config: port=%s poll=%s judge=%v", cfg.Port, cfg.PollInterval, cfg.JudgeEnabled)

	d := service.NewDaemon(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Starting lol-telemetry daemon on ws://localhost:%s/ws\n", cfg.Port)
	fmt.Printf("Judge enabled: %v\n", cfg.JudgeEnabled)

	if err := d.Run(ctx); err != nil {
		fatal("daemon error: %v", err)
	}

	fmt.Println("lol-telemetry daemon stopped")
}
