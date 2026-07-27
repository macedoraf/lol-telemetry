// Package service provides the background daemon and WebSocket API for lol-telemetry.
package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge"
	"lol-telemetry/internal/judge/openrouter"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/orchestrator"
	"lol-telemetry/pkg/riotclient"
)

const defaultBaseURL = "https://127.0.0.1:2999/liveclientdata"

// Daemon runs the background service that polls the Live Client Data API and
// broadcasts game state, events, and judge advice over WebSocket.
type Daemon struct {
	config       DaemonConfig
	client       *riotclient.Client
	hub          *Hub
	orch         *orchestrator.Orchestrator
	pollInterval time.Duration
	lastEvents   int
	connected    bool
	lastErr      string
}

// DaemonConfig holds the runtime configuration for the daemon.
type DaemonConfig struct {
	Port            string
	BaseURL         string
	PollInterval    time.Duration
	JudgeEnabled    bool
	OpenRouterKey   string
	OpenRouterModel string
	Debug           bool
}

// NewDaemon creates a new daemon instance.
func NewDaemon(cfg DaemonConfig) *Daemon {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	client := riotclient.NewClientWithURL(baseURL)
	client.Debug = cfg.Debug

	if baseURL == defaultBaseURL {
		// Some Windows installs bind the API on localhost instead of 127.0.0.1.
		// Try both once at startup.
		if err := client.CheckConnection(); err != nil {
			log.Printf("default LoL API address failed, trying localhost: %v", err)
			alt := riotclient.NewClientWithURL("https://localhost:2999/liveclientdata")
			alt.Debug = cfg.Debug
			if altErr := alt.CheckConnection(); altErr == nil {
				client = alt
				baseURL = "https://localhost:2999/liveclientdata"
				log.Printf("using localhost LoL API address")
			} else {
				log.Printf("localhost LoL API address also failed: %v", altErr)
			}
		}
	}

	hub := NewHub()

	var j *judge.Judge
	if cfg.JudgeEnabled && cfg.OpenRouterKey != "" {
		llmClient := openrouter.NewClientWithModel(cfg.OpenRouterKey, cfg.OpenRouterModel)
		j = judge.NewJudge(llmClient)
	}

	reg := hooks.NewRegistry()
	reg.Register(hooks.Periodic5MinHook{})
	reg.Register(hooks.GameStartHook{})
	reg.Register(hooks.PlayerDeathHook{})
	reg.Register(hooks.RecallHook{})
	reg.Register(hooks.AllyGoldSpikeHook{})
	reg.Register(hooks.EnemyGoldSpikeHook{})
	reg.Register(hooks.FirstTurretHook{})
	reg.Register(hooks.LaningPhaseEndHook{})

	builder := payload.NewBuilder()
	orch := orchestrator.NewOrchestrator(client, reg, builder, j)

	return &Daemon{
		config:       cfg,
		client:       client,
		hub:          hub,
		orch:         orch,
		pollInterval: cfg.PollInterval,
		lastEvents:   -1,
	}
}

// Run starts the WebSocket server and the polling loop.
func (d *Daemon) Run(ctx context.Context) error {
	go d.hub.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(d.hub, w, r)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    ":" + d.config.Port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	go d.pollLoop(ctx)

	log.Printf("lol-telemetry daemon listening on ws://localhost:%s/ws", d.config.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

func (d *Daemon) pollLoop(ctx context.Context) {
	if d.pollInterval <= 0 {
		d.pollInterval = 1 * time.Second
	}
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	d.hub.BroadcastHello()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.tick(ctx)
		}
	}
}

func (d *Daemon) tick(ctx context.Context) {
	data, err := d.client.GetGameData()
	if err != nil {
		if d.connected || d.lastErr != err.Error() {
			log.Printf("LoL API connection failed: %v", err)
			log.Printf("ensure League of Legends is running and you are in an active game; base URL=%s", d.client.BaseURL)
			d.lastErr = err.Error()
		}
		d.connected = false
		d.hub.BroadcastStatus("lcu_error", err.Error())
		return
	}

	if !d.connected {
		log.Printf("LoL API connected: gameTime=%.1f mode=%s", data.GameData.GameTime, data.GameData.GameMode)
		d.connected = true
		d.lastErr = ""
	}

	name, err := d.client.GetActivePlayerName()
	if err != nil {
		name = data.ActivePlayer.SummonerName
	}

	gs := NewGameStateFromAllGameData(data, name)
	if err := d.hub.BroadcastGameState(gs); err != nil {
		log.Printf("broadcast game state error: %v", err)
	}

	d.broadcastEvents(data)

	if d.config.JudgeEnabled {
		responses, err := d.orch.Tick(ctx)
		if err != nil {
			log.Printf("orchestrator tick error: %v", err)
			return
		}
		for _, r := range responses {
			if err := d.hub.BroadcastAdvice(r.HookName, r.GameMinute, r.Advice, r.Reasoning); err != nil {
				log.Printf("broadcast advice error: %v", err)
			}
		}
	}
}

func (d *Daemon) broadcastEvents(data riotclient.AllGameData) {
	if len(data.Events.Events) == 0 {
		return
	}
	if d.lastEvents < 0 {
		d.lastEvents = len(data.Events.Events)
		return
	}
	for i := d.lastEvents; i < len(data.Events.Events); i++ {
		ev := data.Events.Events[i]
		msg := EventMessage{
			EventID:   ev.EventID,
			EventName: ev.EventName,
			EventTime: ev.EventTime,
			Timestamp: time.Now().UnixMilli(),
		}
		if err := d.hub.BroadcastEvent(msg); err != nil {
			log.Printf("broadcast event error: %v", err)
		}
	}
	d.lastEvents = len(data.Events.Events)
}

// CheckConnection attempts a single request to the LoL Live Client Data API and
// reports whether it is reachable and in-game.
func (d *Daemon) CheckConnection() error {
	data, err := d.client.GetGameData()
	if err != nil {
		return err
	}
	if data.GameData.GameTime <= 0 {
		return fmt.Errorf("LoL API reachable but no active game detected (gameTime=%.1f)", data.GameData.GameTime)
	}
	return nil
}

// Config returns the current daemon configuration.
func (d *Daemon) Config() DaemonConfig {
	return d.config
}

// Hub returns the WebSocket hub.
func (d *Daemon) Hub() *Hub {
	return d.hub
}
