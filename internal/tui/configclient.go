package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lol-telemetry/pkg/service"
)

// ConfigClient talks to the daemon's /api/config endpoint.
type ConfigClient struct {
	baseURL string
}

// NewConfigClient derives the HTTP base URL from a WebSocket address.
func NewConfigClient(wsAddr string) *ConfigClient {
	base := strings.Replace(wsAddr, "ws://", "http://", 1)
	base = strings.TrimSuffix(base, "/ws")
	return &ConfigClient{baseURL: base}
}

// Get fetches the full runtime config.
func (c *ConfigClient) Get() (service.ConfigView, error) {
	resp, err := http.Get(c.baseURL + "/api/config")
	if err != nil {
		return service.ConfigView{}, fmt.Errorf("config get: %w", err)
	}
	defer resp.Body.Close()
	var v service.ConfigView
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return service.ConfigView{}, fmt.Errorf("config decode: %w", err)
	}
	return v, nil
}

// Patch sends a partial config update.
func (c *ConfigClient) Patch(patch service.ConfigPatch) (service.ConfigView, error) {
	body, _ := json.Marshal(patch)
	req, err := http.NewRequest(http.MethodPatch, c.baseURL+"/api/config", bytes.NewReader(body))
	if err != nil {
		return service.ConfigView{}, fmt.Errorf("config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return service.ConfigView{}, fmt.Errorf("config patch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp struct{ Error string }
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return service.ConfigView{}, fmt.Errorf("config rejected: %s", errResp.Error)
		}
		return service.ConfigView{}, fmt.Errorf("config rejected (HTTP %d)", resp.StatusCode)
	}
	var v service.ConfigView
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return service.ConfigView{}, fmt.Errorf("config decode: %w", err)
	}
	return v, nil
}
