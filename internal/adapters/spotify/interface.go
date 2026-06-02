// Package spotify defines the port for importing a public Spotify playlist's
// tracklist. The concrete implementation lives in the impl subpackage and a
// test double in mocks.
//
// It relies on the anonymous access token embedded in the open.spotify.com
// playlist embed page; no API keys or Premium account are required.
package spotify

import "context"

// Provider is the Spotify playlist source contract.
type Provider interface {
	// FetchPlaylist resolves a public playlist URL to its name and tracks.
	FetchPlaylist(ctx context.Context, playlistURL string) (Playlist, error)
}

// Playlist is a Spotify playlist: a display name and its ordered tracks.
type Playlist struct {
	Name   string
	Tracks []Track
}

// Track is a single Spotify track, reduced to the fields needed to drive an
// iTunes metadata lookup.
type Track struct {
	Title  string
	Artist string
}
