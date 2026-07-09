// Package store defines the request-cache port and its contract types. The
// concrete in-memory implementation lives in the impl subpackage.
package store

import "audio-scraper/internal/constants"

// Choice is a single selectable search result kept between the /search and
// /download requests.
type Choice struct {
	Type  constants.EntityType `json:"type"`
	ID    string               `json:"id"`
	Label string               `json:"label"`

	// The following carry song identity and metadata between multi-step flows
	// (replacement and curated download).
	Artist       string `json:"artist,omitempty"`
	Album        string `json:"album,omitempty"`
	Track        string `json:"track,omitempty"`
	AlbumArtist  string `json:"album_artist,omitempty"`
	ReleaseDate  string `json:"release_date,omitempty"`
	TrackNumber  int    `json:"track_number,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	URL          string `json:"url,omitempty"`
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

// Provider caches per-request choices with a TTL.
type Provider interface {
	Set(key string, choices Choices)
	Get(key string) (Choices, bool)
	Delete(key string)
	Shutdown()
}
