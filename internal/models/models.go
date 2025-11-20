// Package models defines data models used across the audio scraper service.
package models

import "audio-scraper/internal/constants"

type Choice struct {
	Type  constants.SpotifyEntityType `json:"type"`
	ID    string                      `json:"id"`
	Label string                      `json:"label"`
}

type Choices []Choice

func (choices Choices) FindByLabel(label string) *Choice {
	for _, choice := range choices {
		if choice.Label == label {
			return &choice
		}
	}
	return nil
}

type SearchResponse struct {
	RequestID string   `json:"request_id"`
	Choices   []string `json:"choices"`
}

type DownloadRequest struct {
	RequestID string   `json:"request_id"`
	Choices   []string `json:"choices"`
}

type DownloadJob struct {
	RequestID    string
	TrackID      string
	Track        string
	Album        string
	Artist       string
	ReleaseDate  string
	TrackNumber  int
	ThumbnailURL string
}
