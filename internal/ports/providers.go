package ports

import (
	"context"

	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/models"
)

type SpotifyProvider interface {
	Search(ctx context.Context, query string, t spotify.SearchType, opts ...spotify.RequestOption) (*spotify.SearchResult, error)
}

type StoreProvider interface {
	Set(key string, choices []models.Choice)
	Get(key string) ([]models.Choice, bool)
	Delete(key string)
}
