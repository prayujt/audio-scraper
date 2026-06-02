// Package subsonicimpl implements the subsonic.Provider port against a
// Subsonic-compatible REST API (Navidrome) using salted-token authentication.
//
// If the configured base URL is empty the client is considered disabled: every
// method logs a warning and returns a zero value without making a request. This
// lets callers invoke it unconditionally when no server is configured.
package subsonicimpl

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"audio-scraper/internal/adapters/subsonic"
	"audio-scraper/internal/logger"
)

const (
	// apiVersion is the Subsonic API version we advertise.
	apiVersion = "1.16.1"
	// clientName identifies this client to the server (the "c" param).
	clientName = "audio-scraper"
)

// Client is the Subsonic-backed provider.
type Client struct {
	http     *http.Client
	baseURL  string
	user     string
	password string
}

// compile-time assertion that Client satisfies the port.
var _ subsonic.Provider = (*Client)(nil)

// New returns a Subsonic provider. A blank baseURL disables the client: all
// calls become warning-logged no-ops.
func New(baseURL, user, password string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   3 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          20,
				MaxIdleConnsPerHost:   20,
				ResponseHeaderTimeout: 10 * time.Second,
				TLSHandshakeTimeout:   3 * time.Second,
			},
		},
		baseURL:  strings.TrimRight(baseURL, "/"),
		user:     user,
		password: password,
	}
}

// enabled reports whether a server is configured. When it is not, it logs a
// warning naming the skipped operation and returns false.
func (c *Client) enabled(ctx context.Context, op string) bool {
	if c.baseURL == "" {
		logger.From(ctx).Warn("subsonic url not configured, skipping operation", "op", op)
		return false
	}
	return true
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.enabled(ctx, "ping") {
		return nil
	}
	_, err := c.do(ctx, "ping", nil)
	return err
}

func (c *Client) StartScan(ctx context.Context) error {
	if !c.enabled(ctx, "startScan") {
		return nil
	}
	log := logger.From(ctx)
	log.Info("triggering subsonic library rescan")
	_, err := c.do(ctx, "startScan", nil)
	if err != nil {
		log.Error("subsonic rescan failed", "error", err)
	}
	return err
}

func (c *Client) Search(ctx context.Context, query string) (subsonic.SearchResult, error) {
	if !c.enabled(ctx, "search") {
		return subsonic.SearchResult{}, nil
	}
	resp, err := c.do(ctx, "search3", url.Values{"query": {query}})
	if err != nil {
		return subsonic.SearchResult{}, err
	}
	if resp.SearchResult3 == nil {
		return subsonic.SearchResult{}, nil
	}

	var out subsonic.SearchResult
	for _, a := range resp.SearchResult3.Artist {
		out.Artists = append(out.Artists, subsonic.Artist{ID: a.ID, Name: a.Name})
	}
	for _, a := range resp.SearchResult3.Album {
		out.Albums = append(out.Albums, subsonic.Album{ID: a.ID, Name: a.Name, Artist: a.Artist})
	}
	for _, s := range resp.SearchResult3.Song {
		out.Songs = append(out.Songs, subsonic.Song{
			ID:       s.ID,
			Title:    s.Title,
			Album:    s.Album,
			Artist:   s.Artist,
			Duration: s.Duration,
		})
	}
	return out, nil
}

func (c *Client) GetPlaylists(ctx context.Context) ([]subsonic.Playlist, error) {
	if !c.enabled(ctx, "getPlaylists") {
		return nil, nil
	}
	resp, err := c.do(ctx, "getPlaylists", nil)
	if err != nil {
		return nil, err
	}
	if resp.Playlists == nil {
		return nil, nil
	}
	out := make([]subsonic.Playlist, 0, len(resp.Playlists.Playlist))
	for _, p := range resp.Playlists.Playlist {
		out = append(out, subsonic.Playlist{ID: p.ID, Name: p.Name, SongCount: p.SongCount})
	}
	return out, nil
}

func (c *Client) CreatePlaylist(ctx context.Context, name string, songIDs []string) (subsonic.Playlist, error) {
	if !c.enabled(ctx, "createPlaylist") {
		return subsonic.Playlist{}, nil
	}
	params := url.Values{"name": {name}}
	for _, id := range songIDs {
		params.Add("songId", id)
	}
	resp, err := c.do(ctx, "createPlaylist", params)
	if err != nil {
		return subsonic.Playlist{}, err
	}
	if resp.Playlist == nil {
		return subsonic.Playlist{}, nil
	}
	return subsonic.Playlist{
		ID:        resp.Playlist.ID,
		Name:      resp.Playlist.Name,
		SongCount: resp.Playlist.SongCount,
	}, nil
}

func (c *Client) UpdatePlaylist(ctx context.Context, playlistID string, songIDsToAdd []string) error {
	if !c.enabled(ctx, "updatePlaylist") {
		return nil
	}
	params := url.Values{"playlistId": {playlistID}}
	for _, id := range songIDsToAdd {
		params.Add("songIdToAdd", id)
	}
	_, err := c.do(ctx, "updatePlaylist", params)
	return err
}

func (c *Client) Scrobble(ctx context.Context, songID string) error {
	if !c.enabled(ctx, "scrobble") {
		return nil
	}
	_, err := c.do(ctx, "scrobble", url.Values{"id": {songID}})
	return err
}

func (c *Client) Star(ctx context.Context, songID string) error {
	if !c.enabled(ctx, "star") {
		return nil
	}
	_, err := c.do(ctx, "star", url.Values{"id": {songID}})
	return err
}

// do performs an authenticated GET against /rest/<view>.view and returns the
// decoded subsonic-response body, mapping a "failed" status to an error.
func (c *Client) do(ctx context.Context, view string, extra url.Values) (*apiResponse, error) {
	params := c.authParams()
	for k, vs := range extra {
		for _, v := range vs {
			params.Add(k, v)
		}
	}

	u := c.baseURL + "/rest/" + view + ".view?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subsonic: unexpected status %d", res.StatusCode)
	}

	var env apiEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}
	if env.Response.Status != "ok" {
		if env.Response.Error != nil {
			return nil, fmt.Errorf("subsonic: %s (code %d)",
				env.Response.Error.Message, env.Response.Error.Code)
		}
		return nil, fmt.Errorf("subsonic: request failed with status %q", env.Response.Status)
	}
	return &env.Response, nil
}

// authParams builds the salted-token auth query params required on every call.
func (c *Client) authParams() url.Values {
	salt := randSalt()
	sum := md5.Sum([]byte(c.password + salt))
	return url.Values{
		"u": {c.user},
		"t": {hex.EncodeToString(sum[:])},
		"s": {salt},
		"v": {apiVersion},
		"c": {clientName},
		"f": {"json"},
	}
}

// randSalt returns a random hex salt for token auth.
func randSalt() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is effectively fatal; fall back to a timestamp.
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// apiEnvelope wraps every Subsonic response under the "subsonic-response" key.
type apiEnvelope struct {
	Response apiResponse `json:"subsonic-response"`
}

type apiResponse struct {
	Status        string         `json:"status"`
	Version       string         `json:"version"`
	Error         *apiError      `json:"error"`
	SearchResult3 *searchResult3 `json:"searchResult3"`
	Playlists     *playlists     `json:"playlists"`
	Playlist      *playlist      `json:"playlist"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type searchResult3 struct {
	Artist []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
	Album []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Artist string `json:"artist"`
	} `json:"album"`
	Song []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Album    string `json:"album"`
		Artist   string `json:"artist"`
		Duration int    `json:"duration"`
	} `json:"song"`
}

type playlists struct {
	Playlist []playlist `json:"playlist"`
}

type playlist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SongCount int    `json:"songCount"`
}
