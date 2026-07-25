// Package riotclient provides an agnostic HTTP client for the League of Legends
// Live Client Data API. It does not depend on any application-specific code.
package riotclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://127.0.0.1:2999/liveclientdata"

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

// getJSON performs a GET request to the given path and decodes the JSON
// response into dest. It returns an error for non-2xx status codes or
// malformed JSON.
func (c *Client) getJSON(path string, dest any) error {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}
	return nil
}

// GetGameData fetches /allgamedata and returns a typed struct.
// It returns an error for non-2xx status codes or malformed JSON.
func (c *Client) GetGameData() (AllGameData, error) {
	var data AllGameData
	if err := c.getJSON("/allgamedata", &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetGameDataFromURL fetches from a custom base URL, used for tests and mocks.
func (c *Client) GetGameDataFromURL(baseURL string) (AllGameData, error) {
	c.BaseURL = baseURL
	return c.GetGameData()
}

// GetActivePlayerName fetches /activeplayername and returns the player name.
// The Live Client Data API returns the name as a JSON string literal, so the
// body is decoded into a plain Go string.
func (c *Client) GetActivePlayerName() (string, error) {
	var name string
	if err := c.getJSON("/activeplayername", &name); err != nil {
		return "", err
	}
	return name, nil
}

// GetActivePlayerAbilities fetches /activeplayerabilities.
func (c *Client) GetActivePlayerAbilities() (Abilities, error) {
	var data Abilities
	if err := c.getJSON("/activeplayerabilities", &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetActivePlayerRunes fetches /activeplayerrunes.
func (c *Client) GetActivePlayerRunes() (FullRunes, error) {
	var data FullRunes
	if err := c.getJSON("/activeplayerrunes", &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetPlayerList fetches /playerlist.
func (c *Client) GetPlayerList() ([]AllPlayer, error) {
	var data []AllPlayer
	if err := c.getJSON("/playerlist", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetPlayerScores fetches /playerscores?riotId={id}.
func (c *Client) GetPlayerScores(riotId string) (PlayerScores, error) {
	var data PlayerScores
	path := "/playerscores?riotId=" + url.QueryEscape(riotId)
	if err := c.getJSON(path, &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetPlayerSummonerSpells fetches /playersummonerspells?riotId={id}.
func (c *Client) GetPlayerSummonerSpells(riotId string) (SummonerSpells, error) {
	var data SummonerSpells
	path := "/playersummonerspells?riotId=" + url.QueryEscape(riotId)
	if err := c.getJSON(path, &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetPlayerMainRunes fetches /playermainrunes?riotId={id}.
func (c *Client) GetPlayerMainRunes(riotId string) (PlayerRunes, error) {
	var data PlayerRunes
	path := "/playermainrunes?riotId=" + url.QueryEscape(riotId)
	if err := c.getJSON(path, &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetPlayerItems fetches /playeritems?riotId={id}.
func (c *Client) GetPlayerItems(riotId string) ([]Item, error) {
	var data []Item
	path := "/playeritems?riotId=" + url.QueryEscape(riotId)
	if err := c.getJSON(path, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetEventData fetches /eventdata.
func (c *Client) GetEventData() (Events, error) {
	var data Events
	if err := c.getJSON("/eventdata", &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetGameStats fetches /gamestats.
func (c *Client) GetGameStats() (GameData, error) {
	var data GameData
	if err := c.getJSON("/gamestats", &data); err != nil {
		return data, err
	}
	return data, nil
}
