// Package subsonic defines the port for talking to a Subsonic-compatible
// server (e.g. Navidrome): trigger library rescans, search, and manage
// playlists. The concrete implementation lives in the impl subpackage and a
// test double in mocks.
//
// When the server URL is not configured, the implementation no-ops every call
// and emits a warning, so the rest of the app can call it unconditionally.
package subsonic

import "context"

// Provider is the Subsonic server contract.
type Provider interface {
	// Ping verifies connectivity and credentials.
	Ping(ctx context.Context) error
	// StartScan triggers a library rescan so newly added files are indexed.
	StartScan(ctx context.Context) error
	// Search returns artists, albums and songs matching the query (search3).
	Search(ctx context.Context, query string) (SearchResult, error)
	// GetPlaylists lists the current user's playlists.
	GetPlaylists(ctx context.Context) ([]Playlist, error)
	// CreatePlaylist creates a playlist with the given name and songs, and
	// returns it.
	CreatePlaylist(ctx context.Context, name string, songIDs []string) (Playlist, error)
	// UpdatePlaylist appends the given songs to an existing playlist.
	UpdatePlaylist(ctx context.Context, playlistID string, songIDsToAdd []string) error
	// Scrobble registers a play for a song.
	Scrobble(ctx context.Context, songID string) error
	// Star marks a song as a favorite.
	Star(ctx context.Context, songID string) error
}

// Song is a single track in the Subsonic library.
type Song struct {
	ID       string
	Title    string
	Album    string
	Artist   string
	Duration int
}

// Album is an album entry from a Subsonic search.
type Album struct {
	ID     string
	Name   string
	Artist string
}

// Artist is an artist entry from a Subsonic search.
type Artist struct {
	ID   string
	Name string
}

// Playlist is a Subsonic playlist (metadata only).
type Playlist struct {
	ID        string
	Name      string
	SongCount int
}

// SearchResult aggregates the entity kinds returned by search3.
type SearchResult struct {
	Artists []Artist
	Albums  []Album
	Songs   []Song
}
