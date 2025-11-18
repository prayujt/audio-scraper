package providers

import (
	"audio-scraper/internal/ports"
)

type spotifyClient struct {
	log ports.Logger
}

func NewSpotifyClient(l ports.Logger) ports.SpotifyProvider {
	return &spotifyClient{log: l}
}
