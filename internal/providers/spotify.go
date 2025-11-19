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

func NewSpotifyProvider(l ports.Logger, clientID string, clientSecret string) (ports.SpotifyProvider, error) {
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

func (s *spotifyClient) GetTrack(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.FullTrack, error) {
	log := logger.From(ctx)
	log.Info("fetching spotify track", "track_id", id)
	client := s.client.(*spotify.Client)
	return client.GetTrack(ctx, id, opts...)
}

func (s *spotifyClient) GetAlbum(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.FullAlbum, error) {
	log := logger.From(ctx)
	log.Info("fetching spotify album", "album_id", id)
	client := s.client.(*spotify.Client)
	return client.GetAlbum(ctx, id, opts...)
}

func (s *spotifyClient) GetArtist(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.SimpleAlbumPage, error) {
	log := logger.From(ctx)
	log.Info("fetching spotify artist", "artist_id", id)
	client := s.client.(*spotify.Client)
	albumTypes := []spotify.AlbumType{spotify.AlbumTypeAlbum, spotify.AlbumTypeSingle, spotify.AlbumTypeAppearsOn, spotify.AlbumTypeCompilation}
	return client.GetArtistAlbums(ctx, id, albumTypes, opts...)
}
