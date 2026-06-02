// Package lrclibimpl implements the lrclib.Provider port using the lrclib.net
// public lyrics API.
package lrclibimpl

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"audio-scraper/internal/adapters/lrclib"
	"audio-scraper/internal/logger"
)

const lrclibBaseURL = "https://lrclib.net/api/get"

// Client is the lrclib-backed lyrics provider.
type Client struct {
	http *http.Client
}

var _ lrclib.Provider = (*Client)(nil)

// New returns a lrclib lyrics provider.
func New() *Client {
	client := &http.Client{
		Timeout: 5 * time.Second,
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
	}

	return &Client{
		http: client,
	}
}

func (l *Client) FindLyrics(ctx context.Context, req *lrclib.LrclibRequest) (*lrclib.LRC, error) {
	log := logger.From(ctx)
	log.Info("starting lrclib search")
	params := url.Values{}
	params.Set("artist_name", req.Artist)
	params.Set("track_name", req.Track)
	params.Set("album_name", req.Album)
	params.Set("duration", fmt.Sprintf("%d", req.Duration))

	endpoint := fmt.Sprintf("%s?%s", lrclibBaseURL, params.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	res, err := l.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib: no lyrics found (%d)", res.StatusCode)
	}

	var out lrclibResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out.toLRC(), nil
}

type lrclibResponse struct {
	ID           int     `json:"id"`
	ArtistName   string  `json:"artistName"`
	TrackName    string  `json:"trackName"`
	AlbumName    string  `json:"albumName"`
	Duration     float32 `json:"duration"`
	SyncedLyrics string  `json:"syncedLyrics"`
	PlainLyrics  string  `json:"plainLyrics"`
}

func (r *lrclibResponse) toLRC() *lrclib.LRC {
	lrc := &lrclib.LRC{
		Segments: nil,
		Full:     &r.PlainLyrics,
	}
	if strings.TrimSpace(r.SyncedLyrics) == "" {
		return lrc
	}

	re := regexp.MustCompile(`^\[(\d{2}:\d{2}(?:\.\d{1,3})?)\]\s*(.*)$`)

	lines := strings.Split(r.SyncedLyrics, "\n")
	segments := make([]lrclib.LRCSegment, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		m := re.FindStringSubmatch(line)
		if len(m) != 3 {
			// line doesn't match [mm:ss.xx] pattern – skip it.
			continue
		}

		segments = append(segments, lrclib.LRCSegment{
			Time: m[1],
			Text: m[2],
		})
	}
	if len(segments) == 0 {
		return lrc
	}

	lrc.Segments = segments
	return lrc
}
