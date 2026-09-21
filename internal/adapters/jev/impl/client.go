// Package jevimpl calls the Jev/TypeSafe bounded decision API.
package jevimpl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"audio-scraper/internal/adapters/jev"
)

type Client struct {
	apiKey        string
	endpoint      string
	model         string
	minConfidence float64
	httpClient    *http.Client
}

var _ jev.Ranker = (*Client)(nil)

func New(apiKey, endpoint, model string, minConfidence float64) *Client {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://api.typesafe.ai/v1/systemone"
	}
	if strings.TrimSpace(model) == "" {
		model = "jev-latest"
	}
	if minConfidence <= 0 || minConfidence > 1 {
		minConfidence = 0.80
	}
	return &Client{apiKey: strings.TrimSpace(apiKey), endpoint: endpoint, model: model, minConfidence: minConfidence, httpClient: &http.Client{Timeout: 4 * time.Second}}
}

type request struct {
	Model     string              `json:"model"`
	State     string              `json:"state"`
	Questions map[string]question `json:"questions"`
}
type question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}
type response struct {
	Answers map[string]answer `json:"answers"`
}
type answer struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

func (c *Client) Rank(ctx context.Context, track, artist string, duration int, candidates []jev.Candidate) (string, float64, error) {
	if c.apiKey == "" {
		return "", 0, fmt.Errorf("Jev API key is not configured")
	}
	if len(candidates) < 2 {
		return "", 0, fmt.Errorf("at least two candidates are required")
	}
	criteria := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		criteria[candidate.Key] = fmt.Sprintf("title=%q; uploader=%q; duration=%ds", candidate.Title, candidate.Uploader, candidate.Duration)
	}
	payload := request{Model: c.model, State: fmt.Sprintf("Choose the safest canonical source for track=%q artist=%q expected_duration=%ds. Candidates were pre-filtered deterministically for unsafe alternate types and large duration mismatches.", track, artist, duration), Questions: map[string]question{"best_source": {Type: "choice", Instructions: "Choose the candidate most likely to be the intended canonical release. Prefer artist/label/Topic provenance and the closest duration. Do not invent a source outside the supplied keys.", Criteria: criteria}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("Jev request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("Jev HTTP %d", resp.StatusCode)
	}
	var decoded response
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", 0, fmt.Errorf("decode Jev response: %w", err)
	}
	answer, ok := decoded.Answers["best_source"]
	if !ok || answer.Choice == "" {
		return "", 0, fmt.Errorf("Jev response omitted best_source")
	}
	if answer.Confidence < c.minConfidence {
		return "", answer.Confidence, fmt.Errorf("Jev confidence %.2f below threshold %.2f", answer.Confidence, c.minConfidence)
	}
	for _, candidate := range candidates {
		if candidate.Key == answer.Choice {
			return answer.Choice, answer.Confidence, nil
		}
	}
	return "", answer.Confidence, fmt.Errorf("Jev returned unknown candidate key")
}
