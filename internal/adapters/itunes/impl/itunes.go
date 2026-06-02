// Package itunesimpl implements the itunes.Provider port using Apple's public
// iTunes Search API. The API requires no authentication.
package itunesimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"audio-scraper/internal/adapters/itunes"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
)

const (
	searchURL = "https://itunes.apple.com/search"
	lookupURL = "https://itunes.apple.com/lookup"

	// searchLimit caps results per entity kind; processSearchData trims further.
	searchLimit = 15

	// userAgent is sent to be a well-behaved client of the public API.
	userAgent = "audio-scraper/1.0 (+https://github.com/audio-scraper)"
)

// Client is the iTunes-backed metadata provider.
type Client struct {
	http *http.Client
}

// compile-time assertion that Client satisfies the port.
var _ itunes.Provider = (*Client)(nil)

// New returns an iTunes metadata provider.
func New() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   3 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          50,
				MaxIdleConnsPerHost:   50,
				ResponseHeaderTimeout: 5 * time.Second,
				TLSHandshakeTimeout:   3 * time.Second,
			},
		},
	}
}

// itunesResult is a single entry in an iTunes API response. The same shape is
// reused for tracks, collections (albums) and artists; only a subset of fields
// is populated depending on wrapperType.
type itunesResult struct {
	WrapperType     string `json:"wrapperType"`
	ArtistID        int64  `json:"artistId"`
	CollectionID    int64  `json:"collectionId"`
	TrackID         int64  `json:"trackId"`
	ArtistName      string `json:"artistName"`
	CollectionName  string `json:"collectionName"`
	TrackName       string `json:"trackName"`
	ReleaseDate     string `json:"releaseDate"`
	TrackNumber     int    `json:"trackNumber"`
	TrackTimeMillis int    `json:"trackTimeMillis"`
	ArtworkURL100   string `json:"artworkUrl100"`
}

type itunesResponse struct {
	ResultCount int            `json:"resultCount"`
	Results     []itunesResult `json:"results"`
}

func (c *Client) Search(ctx context.Context, query string) (models.SearchResult, error) {
	log := logger.From(ctx)
	log.Info("performing itunes search", "query", query)

	var (
		result models.SearchResult
		wg     sync.WaitGroup
		mu     sync.Mutex
		errs   error
	)
	record := func(err error) {
		mu.Lock()
		errs = errors.Join(errs, err)
		mu.Unlock()
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		resp, err := c.get(ctx, searchURL, url.Values{
			"term":   {query},
			"media":  {"music"},
			"entity": {"song"},
			"limit":  {strconv.Itoa(searchLimit)},
		})
		if err != nil {
			record(err)
			return
		}
		for _, r := range resp.Results {
			if r.WrapperType == "track" {
				result.Tracks = append(result.Tracks, toTrack(r))
			}
		}
	}()
	go func() {
		defer wg.Done()
		resp, err := c.get(ctx, searchURL, url.Values{
			"term":   {query},
			"media":  {"music"},
			"entity": {"album"},
			"limit":  {strconv.Itoa(searchLimit)},
		})
		if err != nil {
			record(err)
			return
		}
		for _, r := range resp.Results {
			if r.WrapperType == "collection" {
				result.Albums = append(result.Albums, toAlbum(r))
			}
		}
	}()
	go func() {
		defer wg.Done()
		resp, err := c.get(ctx, searchURL, url.Values{
			"term":   {query},
			"media":  {"music"},
			"entity": {"musicArtist"},
			"limit":  {strconv.Itoa(searchLimit)},
		})
		if err != nil {
			record(err)
			return
		}
		for _, r := range resp.Results {
			if r.WrapperType == "artist" {
				result.Artists = append(result.Artists, toArtist(r))
			}
		}
	}()
	wg.Wait()

	if errs != nil {
		log.Error("itunes search failed", "error", errs)
		return models.SearchResult{}, errors.New("itunes search failed")
	}
	return result, nil
}

func (c *Client) GetTrack(ctx context.Context, id string) (models.Track, error) {
	log := logger.From(ctx)
	log.Info("fetching itunes track", "track_id", id)

	resp, err := c.get(ctx, lookupURL, url.Values{"id": {id}, "entity": {"song"}})
	if err != nil {
		return models.Track{}, err
	}
	for _, r := range resp.Results {
		if r.WrapperType == "track" {
			return toTrack(r), nil
		}
	}
	return models.Track{}, fmt.Errorf("itunes: track %s not found", id)
}

func (c *Client) GetAlbum(ctx context.Context, id string) (models.Album, error) {
	log := logger.From(ctx)
	log.Info("fetching itunes album", "album_id", id)

	resp, err := c.get(ctx, lookupURL, url.Values{"id": {id}, "entity": {"song"}})
	if err != nil {
		return models.Album{}, err
	}

	var album models.Album
	found := false
	for _, r := range resp.Results {
		switch r.WrapperType {
		case "collection":
			album = toAlbum(r)
			found = true
		case "track":
			album.Tracks = append(album.Tracks, toTrack(r))
		}
	}
	if !found {
		return models.Album{}, fmt.Errorf("itunes: album %s not found", id)
	}
	return album, nil
}

func (c *Client) GetArtist(ctx context.Context, id string) (models.Artist, error) {
	log := logger.From(ctx)
	log.Info("fetching itunes artist", "artist_id", id)

	resp, err := c.get(ctx, lookupURL, url.Values{"id": {id}, "entity": {"album"}})
	if err != nil {
		return models.Artist{}, err
	}

	var artist models.Artist
	found := false
	for _, r := range resp.Results {
		switch r.WrapperType {
		case "artist":
			artist = toArtist(r)
			found = true
		case "collection":
			artist.Albums = append(artist.Albums, toAlbum(r))
		}
	}
	if !found {
		return models.Artist{}, fmt.Errorf("itunes: artist %s not found", id)
	}
	return artist, nil
}

// get performs a GET against an iTunes endpoint and decodes the response.
func (c *Client) get(ctx context.Context, endpoint string, params url.Values) (*itunesResponse, error) {
	u := endpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itunes: unexpected status %d", res.StatusCode)
	}

	var out itunesResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func toTrack(r itunesResult) models.Track {
	return models.Track{
		ID:          strconv.FormatInt(r.TrackID, 10),
		Name:        r.TrackName,
		Album:       r.CollectionName,
		Artist:      r.ArtistName,
		ReleaseDate: r.ReleaseDate,
		TrackNumber: r.TrackNumber,
		Duration:    r.TrackTimeMillis / 1000,
		ArtworkURL:  upscaleArtwork(r.ArtworkURL100),
	}
}

func toAlbum(r itunesResult) models.Album {
	return models.Album{
		ID:     strconv.FormatInt(r.CollectionID, 10),
		Name:   r.CollectionName,
		Artist: r.ArtistName,
	}
}

func toArtist(r itunesResult) models.Artist {
	return models.Artist{
		ID:   strconv.FormatInt(r.ArtistID, 10),
		Name: r.ArtistName,
	}
}

// upscaleArtwork swaps the default 100x100 thumbnail for a larger variant by
// rewriting the size segment in the artwork URL.
func upscaleArtwork(u string) string {
	if u == "" {
		return ""
	}
	return strings.Replace(u, "100x100bb", "600x600bb", 1)
}
