// Package constants contains constant values used across the application.
package constants

const DownloadWorkerPoolSize = 5

type EntityType string

const (
	EntityTypeTrack  EntityType = "track"
	EntityTypeAlbum  EntityType = "album"
	EntityTypeArtist EntityType = "artist"
)
