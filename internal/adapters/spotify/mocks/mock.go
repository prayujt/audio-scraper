// Package spotifymock provides a test double for spotify.Provider.
package spotifymock

import (
	"context"

	"audio-scraper/internal/adapters/spotify"
)

// Mock implements spotify.Provider; set the *Func fields to control behavior.
type Mock struct {
	FetchPlaylistFunc func(ctx context.Context, playlistURL string) (spotify.Playlist, error)
}

var _ spotify.Provider = (*Mock)(nil)

func (m *Mock) FetchPlaylist(ctx context.Context, playlistURL string) (spotify.Playlist, error) {
	if m.FetchPlaylistFunc != nil {
		return m.FetchPlaylistFunc(ctx, playlistURL)
	}
	return spotify.Playlist{}, nil
}
