package recorder

import "encoding/json"

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
	Version   int    `json:"v"`
	Type      string `json:"type"`
	Ts        int64  `json:"ts"`
	Session   string `json:"session"`
	GameTime  float64 `json:"gameTime"`
	GameMinute int    `json:"gameMinute"`
	HookName  string `json:"hookName"`
	Question  string `json:"question"`
	Advice    string `json:"advice"`
	Reasoning string `json:"reasoning"`
}
