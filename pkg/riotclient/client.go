// Package riotclient provides an agnostic HTTP client for the League of Legends
// Live Client Data API. It does not depend on any application-specific code.
package riotclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://127.0.0.1:2999/liveclientdata"

// AllGameData is the root DTO returned by /allgamedata.
type AllGameData struct {
	ActivePlayer ActivePlayer `json:"activePlayer"`
	AllPlayers   []AllPlayer  `json:"allPlayers"`
	Events       Events       `json:"events"`
	GameData     GameData     `json:"gameData"`
}

// ActivePlayer represents the local active player.
type ActivePlayer struct {
	SummonerName string  `json:"summonerName"`
	Level        int     `json:"level"`
	CurrentGold  float64 `json:"currentGold"`
	ChampionName string  `json:"championName"`
	Scores       Scores  `json:"scores"`
}

// Scores holds the local player minion/monster kill counters.
type Scores struct {
	CreepScore           int `json:"creepScore"`
	NeutralMinionsKilled int `json:"neutralMinionsKilled"`
}

// AllPlayer represents a player in the match.
type AllPlayer struct {
	SummonerName string  `json:"summonerName"`
	Team         string  `json:"team"`
	Level        int     `json:"level"`
	ChampionName string  `json:"championName"`
	IsBot        bool    `json:"isBot"`
	Scores       Scores  `json:"scores"`
}

// Events is a container for the game event list.
type Events struct {
	Events []Event `json:"Events"`
}

// Event is a single game event.
type Event struct {
	EventID   int    `json:"EventID"`
	EventName string `json:"EventName"`
	EventTime float64 `json:"EventTime"`
}

// GameData contains match metadata.
type GameData struct {
	GameTime float64 `json:"gameTime"`
	MapName  string  `json:"mapName"`
	MapNumber int    `json:"mapNumber"`
}

// Client is a minimal Live Client Data API client.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a client configured with SSL verification disabled,
// matching the self-signed certificate served by the local LoL client.
func NewClient() *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Client{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
	}
}

// GetGameData fetches /allgamedata and returns a typed struct.
// It returns an error for non-2xx status codes or malformed JSON.
func (c *Client) GetGameData() (AllGameData, error) {
	var data AllGameData
	url := c.BaseURL + "/allgamedata"
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return data, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return data, fmt.Errorf("decode failed: %w", err)
	}
	return data, nil
}

// GetGameDataFromURL fetches from a custom base URL, used for tests and mocks.
func (c *Client) GetGameDataFromURL(baseURL string) (AllGameData, error) {
	c.BaseURL = baseURL
	return c.GetGameData()
}
