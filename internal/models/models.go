// Package models defines data models used across the audio scraper service.
package models

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
