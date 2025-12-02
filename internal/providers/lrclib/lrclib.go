package lrclib

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"audio-scraper/internal/logger"
)

const lrclibBaseURL = "https://lrclib.net/api/get"

type LrclibClient struct {
	client *http.Client
}

func NewLrclibProvider() *LrclibClient {
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

	return &LrclibClient{
		client: client,
	}
}

func (l *LrclibClient) FindLyrics(ctx context.Context, req *LrclibRequest) (*LRC, error) {
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
