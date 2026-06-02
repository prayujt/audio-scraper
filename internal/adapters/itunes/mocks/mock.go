// Package itunesmock provides a test double for itunes.Provider.
package itunesmock

import (
	"context"

	"audio-scraper/internal/adapters/itunes"
	"audio-scraper/internal/models"
)

// Mock implements itunes.Provider; set the *Func fields to control behavior.
type Mock struct {
	SearchFunc    func(ctx context.Context, query string) (models.SearchResult, error)
	GetTrackFunc  func(ctx context.Context, id string) (models.Track, error)
	GetAlbumFunc  func(ctx context.Context, id string) (models.Album, error)
	GetArtistFunc func(ctx context.Context, id string) (models.Artist, error)
}

var _ itunes.Provider = (*Mock)(nil)

func (m *Mock) Search(ctx context.Context, query string) (models.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query)
	}
	return models.SearchResult{}, nil
}

func (m *Mock) GetTrack(ctx context.Context, id string) (models.Track, error) {
	if m.GetTrackFunc != nil {
		return m.GetTrackFunc(ctx, id)
	}
	return models.Track{}, nil
}

func (m *Mock) GetAlbum(ctx context.Context, id string) (models.Album, error) {
	if m.GetAlbumFunc != nil {
		return m.GetAlbumFunc(ctx, id)
	}
	return models.Album{}, nil
}

func (m *Mock) GetArtist(ctx context.Context, id string) (models.Artist, error) {
	if m.GetArtistFunc != nil {
		return m.GetArtistFunc(ctx, id)
	}
	return models.Artist{}, nil
}
