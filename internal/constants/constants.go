// Package constants contains constant values used across the application.
package constants

type EntityType string

const (
	EntityTypeTrack  EntityType = "track"
	EntityTypeAlbum  EntityType = "album"
	EntityTypeArtist EntityType = "artist"

	// EntityTypeSong is a song already in the Subsonic library (replacement flow).
	EntityTypeSong EntityType = "song"
	// EntityTypeCandidate is a YouTube candidate for a replacement.
	EntityTypeCandidate EntityType = "candidate"
)
