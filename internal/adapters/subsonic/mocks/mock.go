// Package subsonicmock provides a test double for subsonic.Provider.
package subsonicmock

import (
	"context"

	"audio-scraper/internal/adapters/subsonic"
)

// Mock implements subsonic.Provider; set the *Func fields to control behavior.
type Mock struct {
	PingFunc           func(ctx context.Context) error
	StartScanFunc      func(ctx context.Context) error
	SearchFunc         func(ctx context.Context, query string) (subsonic.SearchResult, error)
	GetPlaylistsFunc   func(ctx context.Context) ([]subsonic.Playlist, error)
	CreatePlaylistFunc func(ctx context.Context, name string, songIDs []string) (subsonic.Playlist, error)
	UpdatePlaylistFunc func(ctx context.Context, playlistID string, songIDsToAdd []string) error
	ScrobbleFunc       func(ctx context.Context, songID string) error
	StarFunc           func(ctx context.Context, songID string) error
}

var _ subsonic.Provider = (*Mock)(nil)

func (m *Mock) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func (m *Mock) StartScan(ctx context.Context) error {
	if m.StartScanFunc != nil {
		return m.StartScanFunc(ctx)
	}
	return nil
}

func (m *Mock) Search(ctx context.Context, query string) (subsonic.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query)
	}
	return subsonic.SearchResult{}, nil
}

func (m *Mock) GetPlaylists(ctx context.Context) ([]subsonic.Playlist, error) {
	if m.GetPlaylistsFunc != nil {
		return m.GetPlaylistsFunc(ctx)
	}
	return nil, nil
}

func (m *Mock) CreatePlaylist(ctx context.Context, name string, songIDs []string) (subsonic.Playlist, error) {
	if m.CreatePlaylistFunc != nil {
		return m.CreatePlaylistFunc(ctx, name, songIDs)
	}
	return subsonic.Playlist{}, nil
}

func (m *Mock) UpdatePlaylist(ctx context.Context, playlistID string, songIDsToAdd []string) error {
	if m.UpdatePlaylistFunc != nil {
		return m.UpdatePlaylistFunc(ctx, playlistID, songIDsToAdd)
	}
	return nil
}

func (m *Mock) Scrobble(ctx context.Context, songID string) error {
	if m.ScrobbleFunc != nil {
		return m.ScrobbleFunc(ctx, songID)
	}
	return nil
}

func (m *Mock) Star(ctx context.Context, songID string) error {
	if m.StarFunc != nil {
		return m.StarFunc(ctx, songID)
	}
	return nil
}
