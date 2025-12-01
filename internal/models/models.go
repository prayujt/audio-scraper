// Package models defines data models used across the audio scraper service.
package models

import (
	"audio-scraper/internal/constants"
	"strings"
)

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
	Duration     int
	ThumbnailURL string
}

type LrclibRequest struct {
	Artist   string
	Album    string
	Track    string
	Duration int
}

type LRCSegment struct {
	Time string
	Text string
}

type LRC struct {
	Segments []LRCSegment
	Full     *string
}

func (l *LRC) SyncedToText() string {
	var b strings.Builder
	for _, s := range l.Segments {
		// Each line: [mm:ss.xx] text
		b.WriteString("[")
		b.WriteString(s.Time)
		b.WriteString("] ")
		b.WriteString(s.Text)
		b.WriteRune('\n')
	}
	return b.String()
}
