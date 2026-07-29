// Package service provides the background daemon and WebSocket API for lol-telemetry.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"lol-telemetry/internal/features"
	"lol-telemetry/internal/hooks"
	"lol-telemetry/internal/judge"
	"lol-telemetry/internal/judge/providers"
	"lol-telemetry/internal/judge/payload"
	"lol-telemetry/internal/orchestrator"
	"lol-telemetry/internal/recorder"
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
	builder      *payload.Builder
	reg          *hooks.Registry
	runtime      *RuntimeConfig
	recorder     *recorder.Recorder
	sessionMgr   recorder.SessionManager
	tracker      *features.Tracker
	pipeline     *features.Pipeline
	pollInterval time.Duration
	lastEvents   int
	lastGameTime float64
	connected    bool
	lastErr      string
}

// DaemonConfig holds the runtime configuration for the daemon.
type DaemonConfig struct {
	Port             string
	BaseURL          string
	PollInterval     time.Duration
	JudgeEnabled     bool
	JudgeProvider    string // openrouter, deepinfra, openai
	OpenRouterKey    string
	OpenRouterModel  string
	Debug            bool
	JudgeLanguage    string // en, pt-BR, es
	RecordEnabled    bool
	RecordingsDir    string
	FeaturesEnabled  bool
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
	if cfg.JudgeEnabled {
		llmClient, err := providers.New(providers.Config{
			Provider:        cfg.JudgeProvider,
			OpenRouterKey:   cfg.OpenRouterKey,
			OpenRouterModel: cfg.OpenRouterModel,
		})
		if err != nil {
			log.Printf("judge provider %q failed: %v; judge disabled", cfg.JudgeProvider, err)
		} else {
			j = judge.NewJudge(llmClient)
		}
	}

	reg := hooks.NewRegistry()
	reg.Register(&hooks.Periodic5MinHook{})
	reg.Register(hooks.GameStartHook{})
	reg.Register(hooks.PlayerDeathHook{})
	reg.Register(&hooks.RecallHook{})
	reg.Register(hooks.AllyGoldSpikeHook{})
	reg.Register(hooks.EnemyGoldSpikeHook{})
	reg.Register(hooks.FirstTurretHook{})
	reg.Register(&hooks.LaningPhaseEndHook{})

	normLang := payload.NormalizeLanguage(cfg.JudgeLanguage)
	if normLang != cfg.JudgeLanguage {
		log.Printf("warning: invalid JUDGE_LANGUAGE=%q, using %q", cfg.JudgeLanguage, normLang)
	}
	builder := payload.NewBuilder(normLang)

	var tracker *features.Tracker
	var pipeline *features.Pipeline
	if cfg.FeaturesEnabled {
		tracker = features.NewTracker()
		pipeline = features.NewPipeline()
	}

	orch := orchestrator.NewOrchestrator(client, reg, builder, j, pipeline, tracker)
	runtime := NewRuntimeConfig(normLang, reg, builder, orch)

	d := &Daemon{
		config:       cfg,
		client:       client,
		hub:          hub,
		orch:         orch,
		builder:      builder,
		reg:          reg,
		runtime:      runtime,
		tracker:      tracker,
		pipeline:     pipeline,
		pollInterval: cfg.PollInterval,
		lastEvents:   -1,
	}

	if cfg.RecordEnabled {
		rec, err := recorder.New(cfg.RecordingsDir)
		if err != nil {
			log.Printf("failed to create recorder: %v; recording disabled", err)
		} else {
			d.recorder = rec
		}
	}

	return d
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
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetConfig(d.runtime)(w, r)
		case http.MethodPatch:
			handlePatchConfig(d.runtime)(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
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

	if d.recorder != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := d.recorder.Close(shutdownCtx); err != nil {
				log.Printf("recorder close error: %v", err)
			}
		}()
	}

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
		if d.recorder != nil {
			d.observeRecording(0, false)
		}
		if d.tracker != nil {
			d.tracker.Reset()
			d.lastGameTime = 0
		}
		return
	}

	gameTime := data.GameData.GameTime

	// Minimal session boundary detector for features when recording is off.
	if d.tracker != nil && gameTime < d.lastGameTime {
		d.tracker.Reset()
		d.lastGameTime = 0
	}

	if !d.connected {
		log.Printf("LoL API connected: gameTime=%.1f mode=%s", gameTime, data.GameData.GameMode)
		d.connected = true
		d.lastErr = ""
	}

	if d.tracker != nil {
		d.tracker.Add(data)
	}

	if d.recorder != nil {
		d.observeRecording(gameTime, true)
		d.recordSnapshot(data)
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

	var responses []orchestrator.Result
	if d.config.JudgeEnabled {
		var err error
		responses, err = d.orch.Tick(ctx)
		if err != nil {
			log.Printf("orchestrator tick error: %v", err)
			return
		}
		for _, r := range responses {
			if err := d.hub.BroadcastAdvice(r.HookName, r.GameMinute, r.Advice, r.Reasoning); err != nil {
				log.Printf("broadcast advice error: %v", err)
			}
			if d.recorder != nil {
				if sid := d.recorder.SessionID(); sid != "" {
					d.recorder.RecordTip(recorder.TipRecord{
						Version:    1,
						Type:       "tip",
						Ts:         time.Now().UnixMilli(),
						Session:    sid,
						GameTime:   r.GameTime,
						GameMinute: r.GameMinute,
						HookName:   r.HookName,
						Question:   r.Question,
						Advice:     r.Advice,
						Reasoning:  r.Reasoning,
					})
				} else {
					log.Printf("recording: tip dropped (no active session)")
				}
			}
		}
	}

	if d.pipeline != nil && d.recorder != nil {
		crossedMinute := int(gameTime/60) > int(d.lastGameTime/60)
		if len(responses) > 0 || crossedMinute {
			d.recordFeatures(gameTime)
		}
	}

	d.lastGameTime = gameTime
}

func (d *Daemon) recordFeatures(gameTime float64) {
	if d.pipeline == nil || d.tracker == nil || d.recorder == nil {
		return
	}
	sid := d.recorder.SessionID()
	if sid == "" {
		return
	}
	fv := d.pipeline.Compute(d.tracker.Window())
	d.recorder.RecordFeature(recorder.FeatureRecord{
		Version:    1,
		Type:       "features",
		Ts:         time.Now().UnixMilli(),
		Session:    sid,
		GameTime:   gameTime,
		GameMinute: fv.GameMinute,
		Features:   fv,
	})
}

func (d *Daemon) observeRecording(gameTime float64, apiOK bool) {
	id, started, ended := d.sessionMgr.Observe(gameTime, apiOK)
	if started || ended {
		if d.tracker != nil {
			d.tracker.Reset()
			d.lastGameTime = 0
		}
	}
	if started && ended {
		prev := d.recorder.SessionID()
		if err := d.recorder.StartSession(id); err != nil {
			log.Printf("recording: failed to start session %s: %v", id, err)
			return
		}
		log.Printf("recording: session %s ended, new session %s started (written=%d dropped=%d)", prev, id, d.recorder.Written(), d.recorder.Dropped())
		return
	}
	if started {
		if err := d.recorder.StartSession(id); err != nil {
			log.Printf("recording: failed to start session %s: %v", id, err)
			return
		}
		log.Printf("recording: session %s started", id)
	}
	if ended {
		d.recorder.EndSession()
		log.Printf("recording: session %s ended (written=%d dropped=%d)", id, d.recorder.Written(), d.recorder.Dropped())
	}
}

func (d *Daemon) recordSnapshot(data riotclient.AllGameData) {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Printf("recording: failed to marshal snapshot: %v", err)
		return
	}
	d.recorder.Record(recorder.TelemetryRecord{
		Version:  1,
		Type:     "telemetry",
		Ts:       time.Now().UnixMilli(),
		Session:  d.recorder.SessionID(),
		GameTime: data.GameData.GameTime,
		Data:     raw,
	})
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
