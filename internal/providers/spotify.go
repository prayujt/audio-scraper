package providers

import (
	"context"
	"errors"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"

	"audio-scraper/internal/logger"
	"audio-scraper/internal/ports"
)

type spotifyClient struct {
	client any
}

func NewSpotifyClient(l ports.Logger, clientID string, clientSecret string) (ports.SpotifyProvider, error) {
	if clientID == "" {
		l.Error("SPOTIFY_CLIENT_ID environment variable is not set")
		return nil, errors.New("missing SPOTIFY_CLIENT_ID")
	}
	if clientSecret == "" {
		l.Error("SPOTIFY_CLIENT_SECRET environment variable is not set")
		return nil, errors.New("missing SPOTIFY_CLIENT_SECRET")
	}
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		l.Error("could not get token", "err", err)
		return nil, err
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)
	return &spotifyClient{client: client}, nil
}

func (s *spotifyClient) Search(ctx context.Context, query string, t spotify.SearchType, opts ...spotify.RequestOption) (*spotify.SearchResult, error) {
	log := logger.From(ctx)
	log.Info("performing spotify search", "query", query, "type", t)
	client := s.client.(*spotify.Client)
	return client.Search(ctx, query, t, opts...)
}
