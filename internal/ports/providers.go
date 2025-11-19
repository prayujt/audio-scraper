package ports

import (
	"context"

	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/models"
)

type SpotifyProvider interface {
	Search(ctx context.Context, query string, t spotify.SearchType, opts ...spotify.RequestOption) (*spotify.SearchResult, error)
	GetTrack(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.FullTrack, error)
	GetAlbum(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.FullAlbum, error)
	GetArtist(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.SimpleAlbumPage, error)
}

type StoreProvider interface {
	Set(key string, choices map[string]models.Choice)
	Get(key string) (map[string]models.Choice, bool)
	Delete(key string)
}
