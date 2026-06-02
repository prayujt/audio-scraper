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

// ChoiceRequest is a single-selection request referencing a previously returned
// choice label (used by the replacement flow's candidate/replace steps).
type ChoiceRequest struct {
	RequestID string `json:"request_id"`
	Choice    string `json:"choice"`
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
	// YouTubeURL, when set, marks this as a replacement job: the audio at this
	// URL replaces the existing file for Track/Album/Artist while preserving
	// the file's existing tags.
	YouTubeURL string
}

// Track is a provider-neutral representation of a single song. It carries the
// flat set of fields required to build a DownloadJob.
type Track struct {
	ID          string
	Name        string
	Album       string
	Artist      string
	ReleaseDate string
	TrackNumber int
	Duration    int
	ArtworkURL  string
}

// Album is a provider-neutral album. Tracks is populated when an album is
// fetched directly (e.g. GetAlbum) and may be empty when the album only appears
// as a search result or within an artist's discography.
type Album struct {
	ID     string
	Name   string
	Artist string
	Tracks []Track
}

// Artist is a provider-neutral artist. Albums carries album-level metadata only
// (no tracks); fetch each album individually to resolve its tracks.
type Artist struct {
	ID     string
	Name   string
	Albums []Album
}

// SearchResult aggregates the three entity kinds returned by a metadata search.
type SearchResult struct {
	Tracks  []Track
	Albums  []Album
	Artists []Artist
}
