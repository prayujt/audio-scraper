package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

type youtubeClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewYTProvider(l ports.Logger, apiKey string) (ports.YTProvider, error) {
	if apiKey == "" {
		l.Error("GOOGLE_API_KEY environment variable is not set")
		return nil, errors.New("missing GOOGLE_API_KEY")
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &youtubeClient{
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
}

func (y *youtubeClient) Search(ctx context.Context, query string, maxResults int) ([]models.YTSearchResult, error) {
	log := logger.From(ctx)
	if maxResults <= 0 {
		maxResults = 5
	}

	log.Info("performing youtube search", "query", query, "max_results", maxResults)

	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("type", "video")
	params.Set("q", query)
	params.Set("maxResults", strconv.Itoa(maxResults))
	params.Set("key", y.apiKey)

	u := constants.YTSearchEndpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		log.Error("failed to create youtube search request", "err", err)
		return nil, err
	}

	resp, err := y.httpClient.Do(req)
	if err != nil {
		log.Error("youtube search request failed", "err", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("yt search returned non-200 status",
			"status_code", resp.StatusCode)
		return nil, fmt.Errorf("yt search failed with status %d", resp.StatusCode)
	}

	var result models.YTSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Error("failed to decode yt search response", "err", err)
		return nil, err
	}

	log.Info("yt search completed", "items", len(result.Items))

	var results []models.YTSearchResult
	for _, item := range result.Items {
		results = append(results, models.YTSearchResult{
			VideoID:      item.ID.VideoID,
			Title:        item.Snippet.Title,
			ThumbnailURL: item.Snippet.Thumbnails.Default.URL,
		})
	}
	return results, nil
}
