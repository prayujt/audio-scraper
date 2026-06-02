// Package spotifyimpl implements the spotify.Provider port without any API
// keys. It scrapes the anonymous access token and the first page of tracks
// from the open.spotify.com playlist embed page, then uses that token to page
// the full tracklist via Spotify's internal pathfinder GraphQL endpoint.
//
// FRAGILITY: the pathfinder persisted-query hash (fetchPlaylistHash) is an
// undocumented client constant that Spotify rotates without notice. When it
// stops matching, the pathfinder call fails and we fall back to the (<=100
// track) list embedded in the page. If full playlists silently truncate to
// 100 tracks, refresh the hash from a current open.spotify.com client bundle.
package spotifyimpl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"audio-scraper/internal/adapters/spotify"
	"audio-scraper/internal/logger"
)

const (
	// fetchPlaylistHash is the persisted-query hash for the fetchPlaylist
	// pathfinder operation. See the package comment on rotation.
	fetchPlaylistHash = "73a3b3470804983e4d55d83cd6cc99715019228fd999d51429cc69473a18789d"

	// pageLimit is the page size the pathfinder API accepts per request.
	pageLimit = 100

	// browserUA mimics a real browser so the embed page serves the full
	// __NEXT_DATA__ payload including the anonymous access token.
	browserUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// nextDataRe extracts the JSON body of the __NEXT_DATA__ script tag.
var nextDataRe = regexp.MustCompile(
	`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

// Client is the token-scraping Spotify provider.
type Client struct {
	http *http.Client
}

// compile-time assertion that Client satisfies the port.
var _ spotify.Provider = (*Client)(nil)

// New returns a Spotify playlist provider.
func New() *Client {
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
	}
}

func (c *Client) FetchPlaylist(ctx context.Context, playlistURL string) (spotify.Playlist, error) {
	log := logger.From(ctx)

	id, err := parsePlaylistID(playlistURL)
	if err != nil {
		return spotify.Playlist{}, err
	}
	log = log.With("playlist_id", id)
	log.Info("fetching spotify playlist")

	token, embed, err := c.fetchEmbed(ctx, id)
	if err != nil {
		return spotify.Playlist{}, err
	}
	if embed.Name == "" {
		return spotify.Playlist{}, fmt.Errorf("spotify: could not resolve playlist %s", id)
	}

	// Try to page the full tracklist via pathfinder. On any failure fall
	// back to the (<=100) tracks embedded in the page.
	tracks, err := c.fetchAll(ctx, id, token)
	if err != nil {
		log.Warn("pathfinder fetch failed, falling back to embed tracklist",
			"error", err, "embed_tracks", len(embed.Tracks))
		return spotify.Playlist{Name: embed.Name, Tracks: embed.Tracks}, nil
	}

	log.Info("resolved spotify playlist", "name", embed.Name, "tracks", len(tracks))
	return spotify.Playlist{Name: embed.Name, Tracks: tracks}, nil
}

// embedData is the subset of the embed page we care about.
type embedData struct {
	Name   string
	Tracks []spotify.Track
}

// fetchEmbed loads the embed page and extracts the anonymous access token and
// the playlist name + first page of tracks from the __NEXT_DATA__ JSON.
func (c *Client) fetchEmbed(ctx context.Context, id string) (token string, data embedData, err error) {
	u := "https://open.spotify.com/embed/playlist/" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", embedData{}, err
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "text/html")

	res, err := c.http.Do(req)
	if err != nil {
		return "", embedData{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", embedData{}, fmt.Errorf("spotify: embed page status %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", embedData{}, err
	}

	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return "", embedData{}, fmt.Errorf("spotify: __NEXT_DATA__ not found in embed page")
	}
	var raw any
	if err := json.Unmarshal(m[1], &raw); err != nil {
		return "", embedData{}, fmt.Errorf("spotify: decode __NEXT_DATA__: %w", err)
	}

	token = findAccessToken(raw)
	data = extractEmbedData(raw)
	return token, data, nil
}

// fetchAll pages the complete tracklist via the pathfinder endpoint.
func (c *Client) fetchAll(ctx context.Context, id, token string) ([]spotify.Track, error) {
	if token == "" {
		return nil, fmt.Errorf("spotify: no access token")
	}

	var tracks []spotify.Track
	for offset := 0; ; offset += pageLimit {
		page, total, err := c.fetchPage(ctx, id, token, offset)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, page...)
		if len(page) == 0 || len(tracks) >= total {
			break
		}
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("spotify: pathfinder returned no tracks")
	}
	return tracks, nil
}

// pathfinderResponse models the slice of the fetchPlaylist response we read.
type pathfinderResponse struct {
	Data struct {
		PlaylistV2 struct {
			Content struct {
				TotalCount int `json:"totalCount"`
				Items      []struct {
					ItemV2 struct {
						Data struct {
							Name    string `json:"name"`
							Artists struct {
								Items []struct {
									Profile struct {
										Name string `json:"name"`
									} `json:"profile"`
								} `json:"items"`
							} `json:"artists"`
						} `json:"data"`
					} `json:"itemV2"`
				} `json:"items"`
			} `json:"content"`
		} `json:"playlistV2"`
	} `json:"data"`
}

// fetchPage fetches a single page of tracks and the playlist's total count.
func (c *Client) fetchPage(ctx context.Context, id, token string, offset int) ([]spotify.Track, int, error) {
	variables := fmt.Sprintf(
		`{"uri":"spotify:playlist:%s","offset":%d,"limit":%d}`,
		id, offset, pageLimit)
	extensions := fmt.Sprintf(
		`{"persistedQuery":{"version":1,"sha256Hash":"%s"}}`,
		fetchPlaylistHash)

	q := url.Values{
		"operationName": {"fetchPlaylist"},
		"variables":     {variables},
		"extensions":    {extensions},
	}
	u := "https://api-partner.spotify.com/pathfinder/v1/query?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("spotify: pathfinder status %d", res.StatusCode)
	}

	var pr pathfinderResponse
	if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
		return nil, 0, fmt.Errorf("spotify: decode pathfinder: %w", err)
	}

	content := pr.Data.PlaylistV2.Content
	tracks := make([]spotify.Track, 0, len(content.Items))
	for _, it := range content.Items {
		d := it.ItemV2.Data
		if d.Name == "" {
			continue
		}
		artist := ""
		if len(d.Artists.Items) > 0 {
			artist = d.Artists.Items[0].Profile.Name
		}
		tracks = append(tracks, spotify.Track{Title: d.Name, Artist: artist})
	}
	return tracks, content.TotalCount, nil
}

// playlistIDRe matches the path segment after /playlist/ in any spotify URL.
var playlistIDRe = regexp.MustCompile(`playlist[:/]([A-Za-z0-9]+)`)

// parsePlaylistID extracts the playlist ID from a spotify URL or URI, stripping
// any query string such as ?si=.
func parsePlaylistID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("spotify: empty playlist url")
	}
	m := playlistIDRe.FindStringSubmatch(raw)
	if m == nil {
		return "", fmt.Errorf("spotify: could not parse playlist id from %q", raw)
	}
	return m[1], nil
}

// findAccessToken walks the decoded __NEXT_DATA__ tree for an "accessToken".
func findAccessToken(v any) string {
	switch t := v.(type) {
	case map[string]any:
		if tok, ok := t["accessToken"].(string); ok && tok != "" {
			return tok
		}
		for _, child := range t {
			if tok := findAccessToken(child); tok != "" {
				return tok
			}
		}
	case []any:
		for _, child := range t {
			if tok := findAccessToken(child); tok != "" {
				return tok
			}
		}
	}
	return ""
}

// extractEmbedData walks the tree for the playlist entity carrying a "name"
// and a "trackList" of {title, subtitle} entries.
func extractEmbedData(v any) embedData {
	var out embedData
	var walk func(any)
	walk = func(n any) {
		if out.Name != "" {
			return
		}
		switch t := n.(type) {
		case map[string]any:
			name, hasName := t["name"].(string)
			list, hasList := t["trackList"].([]any)
			if hasName && hasList && name != "" {
				out.Name = name
				for _, item := range list {
					m, ok := item.(map[string]any)
					if !ok {
						continue
					}
					title, _ := m["title"].(string)
					subtitle, _ := m["subtitle"].(string)
					if title == "" {
						continue
					}
					out.Tracks = append(out.Tracks, spotify.Track{
						Title:  title,
						Artist: subtitle,
					})
				}
				return
			}
			for _, child := range t {
				walk(child)
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}
	walk(v)
	return out
}
