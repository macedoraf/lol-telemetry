// Package riotclient provides an agnostic HTTP client for the League of Legends
// Live Client Data API. It does not depend on any application-specific code.
package riotclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const defaultBaseURL = "https://127.0.0.1:2999/liveclientdata"

// Client is a minimal Live Client Data API client.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Debug      bool // Log every request/response when true.
}

// NewClient returns a client configured with SSL verification disabled,
// matching the self-signed certificate served by the local LoL client.
func NewClient() *Client {
	return NewClientWithURL(defaultBaseURL)
}

// NewClientWithURL returns a client configured with SSL verification disabled
// and a custom base URL. This is useful for tests, mocks, or when the LoL
// client is reachable on a different address.
func NewClientWithURL(baseURL string) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Client{
		BaseURL: baseURL,
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
	url := c.BaseURL + path
	if c.Debug {
		log.Printf("[riotclient] GET %s", url)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if c.Debug {
		if dump, err := httputil.DumpRequestOut(req, false); err == nil {
			log.Printf("[riotclient] request dump: %s", string(dump))
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if c.Debug {
			log.Printf("[riotclient] GET %s failed: %v", url, err)
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if c.Debug {
		if dump, err := httputil.DumpResponse(resp, false); err == nil {
			log.Printf("[riotclient] response dump: %s", string(dump))
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if c.Debug {
			log.Printf("[riotclient] GET %s status=%d body=%s", url, resp.StatusCode, string(body))
		}
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.Debug {
			log.Printf("[riotclient] GET %s read body failed: %v", url, err)
		}
		return fmt.Errorf("read body failed: %w", err)
	}
	if c.Debug && len(body) > 0 {
		const maxBody = 2000
		shown := len(body)
		if shown > maxBody {
			shown = maxBody
		}
		log.Printf("[riotclient] GET %s body (truncated): %s", url, string(body[:shown]))
	}

	if err := json.Unmarshal(body, dest); err != nil {
		if c.Debug {
			log.Printf("[riotclient] GET %s decode failed: %v", url, err)
		}
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

// CheckConnection attempts a single /allgamedata request and returns a detailed
// error suitable for troubleshooting. It is safe to call when the game might not
// be running.
func (c *Client) CheckConnection() error {
	_, err := c.GetGameData()
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (is League of Legends running and are you in an active game?)", err)
}

// DiscoverBaseURL tries to reach the LoL Live Client Data API on common
// addresses and returns the first working base URL. If none work, it returns
// an error aggregating all attempts.
func DiscoverBaseURL(timeout time.Duration, debug bool) (string, error) {
	candidates := []string{
		"https://127.0.0.1:2999/liveclientdata",
		"https://localhost:2999/liveclientdata",
	}
	if debug {
		log.Printf("[riotclient] discovering Live Client Data API...")
	}
	var lastErr error
	for _, url := range candidates {
		client := NewClientWithURL(url)
		client.HTTPClient.Timeout = timeout
		client.Debug = debug
		if debug {
			log.Printf("[riotclient] trying %s", url)
		}
		_, err := client.GetGameData()
		if err == nil {
			if debug {
				log.Printf("[riotclient] discovered working URL: %s", url)
			}
			return url, nil
		}
		lastErr = err
		if debug {
			log.Printf("[riotclient] %s failed: %v", url, err)
		}
	}
	return "", fmt.Errorf("could not discover LoL Live Client Data API on common addresses: %w", lastErr)
}
