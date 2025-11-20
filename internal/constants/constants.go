// Package constants contains constant values used across the application.
package constants

const DownloadWorkerPoolSize = 5

type SpotifyEntityType string

const (
	SpotifyEntityTypeTrack  SpotifyEntityType = "track"
	SpotifyEntityTypeAlbum  SpotifyEntityType = "album"
	SpotifyEntityTypeArtist SpotifyEntityType = "artist"
)
