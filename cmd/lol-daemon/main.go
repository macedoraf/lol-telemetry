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

	var check bool
	flag.StringVar(&cfg.Port, "port", cfg.Port, "WebSocket server port")
	flag.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "Live Client Data API polling interval")
	flag.StringVar(&cfg.BaseURL, "lol-url", cfg.BaseURL, "LoL Live Client Data API base URL")
	flag.BoolVar(&check, "check", false, "test the LoL API connection once and exit")
	flag.BoolVar(&cfg.Debug, "debug", cfg.Debug, "enable verbose HTTP logging for the LoL API")
	flag.Parse()

	log.Printf("config: port=%s lolUrl=%s poll=%s judge=%v debug=%v", cfg.Port, cfg.BaseURL, cfg.PollInterval, cfg.JudgeEnabled, cfg.Debug)

	d := service.NewDaemon(cfg)

	if check {
		if err := d.CheckConnection(); err != nil {
			fatal("LoL API connection failed: %v", err)
		}
		fmt.Println("LoL API connection OK")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Starting lol-telemetry daemon on ws://localhost:%s/ws\n", cfg.Port)
	fmt.Printf("Judge enabled: %v\n", cfg.JudgeEnabled)

	if err := d.Run(ctx); err != nil {
		fatal("daemon error: %v", err)
	}

	fmt.Println("lol-telemetry daemon stopped")
}
