// Package constants contains constant values used across the application.
package constants

type SpotifyEntityType string

const (
	SpotifyEntityTypeTrack  SpotifyEntityType = "track"
	SpotifyEntityTypeAlbum  SpotifyEntityType = "album"
	SpotifyEntityTypeArtist SpotifyEntityType = "artist"
)

const DownloadWorkerPoolSize = 5
