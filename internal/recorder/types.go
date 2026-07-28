package recorder

import (
	"encoding/json"

	"lol-telemetry/internal/types"
)

// TelemetryRecord is one persisted snapshot of the raw Live Client Data API.
// It is intentionally a stable envelope so downstream tools can read it easily.
type TelemetryRecord struct {
	Version  int             `json:"v"`
	Type     string          `json:"type"`
	Ts       int64           `json:"ts"`
	Session  string          `json:"session"`
	GameTime float64         `json:"gameTime"`
	Data     json.RawMessage `json:"data"`
}

// TipRecord is one persisted Judge tip correlated to a telemetry session.
type TipRecord struct {
	Version    int     `json:"v"`
	Type       string  `json:"type"`
	Ts         int64   `json:"ts"`
	Session    string  `json:"session"`
	GameTime   float64 `json:"gameTime"`
	GameMinute int     `json:"gameMinute"`
	HookName   string  `json:"hookName"`
	Question   string  `json:"question"`
	Advice     string  `json:"advice"`
	Reasoning  string  `json:"reasoning"`
}

// FeatureRecord is one persisted time-series feature vector correlated to a
// telemetry session. It is written both on trigger fire and at every full
// 60-second game mark.
type FeatureRecord struct {
	Version    int                 `json:"v"`
	Type       string              `json:"type"`
	Ts         int64               `json:"ts"`
	Session    string              `json:"session"`
	GameTime   float64             `json:"gameTime"`
	GameMinute int                 `json:"gameMinute"`
	Features   types.FeatureVector `json:"features"`
}
