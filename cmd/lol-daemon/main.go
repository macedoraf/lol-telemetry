// Command lol-daemon runs the lol-telemetry SDK as a background service.
// It polls the League of Legends Live Client Data API and exposes game state,
// events, and judge advice over WebSocket for external tools like overlays.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"lol-telemetry/pkg/service"
)

func main() {
	var cfg service.DaemonConfig
	flag.StringVar(&cfg.Port, "port", "8080", "WebSocket server port")
	flag.Parse()

	envCfg := service.LoadDaemonConfigFromEnv()
	if !flag.Parsed() {
		cfg = envCfg
	} else {
		// Merge CLI flags with env defaults.
		if cfg.Port == "" {
			cfg.Port = envCfg.Port
		}
		cfg.PollInterval = envCfg.PollInterval
		cfg.JudgeEnabled = envCfg.JudgeEnabled
		cfg.OpenRouterKey = envCfg.OpenRouterKey
		cfg.OpenRouterModel = envCfg.OpenRouterModel
	}

	d := service.NewDaemon(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Starting lol-telemetry daemon on ws://localhost:%s/ws\n", cfg.Port)
	fmt.Printf("Judge enabled: %v\n", cfg.JudgeEnabled)

	if err := d.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "daemon error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("lol-telemetry daemon stopped")
}
