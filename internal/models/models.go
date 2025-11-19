// Package models defines data models used across the audio scraper service.
package models

import "audio-scraper/internal/constants"

type Choice struct {
	Type constants.SpotifyEntityType `json:"type"`
	ID   string                      `json:"id"`
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
	RequestID string
	TrackID   string
	Track     string
	Album     string
	Artist    string
}

type YTSearchResponse struct {
	Kind  string `json:"kind"`
	Etag  string `json:"etag"`
	Items []struct {
		Kind string `json:"kind"`
		Etag string `json:"etag"`
		ID   struct {
			Kind    string `json:"kind"`
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			PublishedAt  string `json:"publishedAt"`
			ChannelID    string `json:"channelId"`
			Title        string `json:"title"`
			Description  string `json:"description"`
			ChannelTitle string `json:"channelTitle"`
			Thumbnails   struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
				High struct {
					URL string `json:"url"`
				} `json:"high"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type YTSearchResult struct {
	VideoID      string `json:"videoId"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnailUrl"`
}
