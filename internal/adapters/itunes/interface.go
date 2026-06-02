// Package itunes defines the metadata provider port backed by the iTunes
// Search API. The concrete implementation lives in the impl subpackage and
// a test double in mocks.
package itunes

import (
	"context"

	"audio-scraper/internal/models"
)

// Provider is the metadata source contract: search for entities and resolve
// them to full details by ID.
type Provider interface {
	// Search returns tracks, albums and artists matching the query.
	Search(ctx context.Context, query string) (models.SearchResult, error)
	// GetTrack resolves a single track by ID.
	GetTrack(ctx context.Context, id string) (models.Track, error)
	// GetAlbum resolves an album by ID with its tracks fully populated.
	GetAlbum(ctx context.Context, id string) (models.Album, error)
	// GetArtist resolves an artist by ID with album-level metadata only.
	GetArtist(ctx context.Context, id string) (models.Artist, error)
}
