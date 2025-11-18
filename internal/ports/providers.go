package ports

import (
	"context"

	"github.com/zmb3/spotify/v2"
)

type SpotifyProvider interface {
	Search(ctx context.Context, query string, t spotify.SearchType, opts ...spotify.RequestOption) (*spotify.SearchResult, error)
}
