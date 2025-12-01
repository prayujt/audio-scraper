package providers

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

	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

const lrclibBaseURL = "https://lrclib.net/api/get"

type lrclibProvider struct {
	client *http.Client
}

func NewLrclibProvider() ports.LrclibProvider {
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

	return &lrclibProvider{
		client: client,
	}
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

func (r *lrclibResponse) ToLRC() *models.LRC {
	lrc := &models.LRC{
		Segments: nil,
		Full:     &r.PlainLyrics,
	}
	if strings.TrimSpace(r.SyncedLyrics) == "" {
		return lrc
	}

	re := regexp.MustCompile(`^\[(\d{2}:\d{2}(?:\.\d{1,3})?)\]\s*(.*)$`)

	lines := strings.Split(r.SyncedLyrics, "\n")
	segments := make([]models.LRCSegment, 0, len(lines))

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

		segments = append(segments, models.LRCSegment{
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

func (l *lrclibProvider) FindLyrics(ctx context.Context, req *models.LrclibRequest) (*models.LRC, error) {
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

	res, err := l.client.Do(httpReq)
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

	return out.ToLRC(), nil
}
