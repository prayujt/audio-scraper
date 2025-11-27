package providers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"

	"audio-scraper/internal/logger"
	"audio-scraper/internal/ports"
)

type spotifyProvider struct {
	config *clientcredentials.Config
	client *spotify.Client

	tokenExpiry time.Time

	mu sync.RWMutex
}

func NewSpotifyProvider(clientID string, clientSecret string) (ports.SpotifyProvider, error) {
	if clientID == "" {
		return nil, errors.New("missing SPOTIFY_CLIENT_ID")
	}
	if clientSecret == "" {
		return nil, errors.New("missing SPOTIFY_CLIENT_SECRET")
	}
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	p := &spotifyProvider{config: config}
	if err := p.refreshClient(context.Background()); err != nil {
		return nil, err
	}
	return p, nil
}

func newSpotifyClient(ctx context.Context, config *clientcredentials.Config) (*spotify.Client, time.Time, error) {
	token, err := config.Token(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)
	return client, token.Expiry, nil
}

func (p *spotifyProvider) refreshClient(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil && time.Now().Before(p.tokenExpiry.Add(-1*time.Minute)) {
		return nil
	}

	client, expiry, err := newSpotifyClient(ctx, p.config)
	if err != nil {
		return err
	}

	p.client = client
	p.tokenExpiry = expiry
	return nil
}

func (p *spotifyProvider) withClient(ctx context.Context) (*spotify.Client, error) {
	p.mu.RLock()
	client := p.client
	expiry := p.tokenExpiry
	p.mu.RUnlock()

	if client == nil || time.Now().After(expiry.Add(-1*time.Minute)) {
		if err := p.refreshClient(ctx); err != nil {
			return nil, err
		}

		p.mu.RLock()
		client = p.client
		p.mu.RUnlock()
	}

	return client, nil
}

func (s *spotifyProvider) Search(ctx context.Context, query string, t spotify.SearchType, opts ...spotify.RequestOption) (*spotify.SearchResult, error) {
	log := logger.From(ctx)
	log.Info("performing spotify search", "query", query, "type", t)
	client, err := s.withClient(ctx)
	if err != nil {
		log.Error("failed to acquire client")
	}
	return client.Search(ctx, query, t, opts...)
}

func (s *spotifyProvider) GetTrack(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.FullTrack, error) {
	log := logger.From(ctx)
	log.Info("fetching spotify track", "track_id", id)
	client, err := s.withClient(ctx)
	if err != nil {
		log.Error("failed to acquire client")
	}
	return client.GetTrack(ctx, id, opts...)
}

func (s *spotifyProvider) GetAlbum(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.FullAlbum, error) {
	log := logger.From(ctx)
	log.Info("fetching spotify album", "album_id", id)
	client, err := s.withClient(ctx)
	if err != nil {
		log.Error("failed to acquire client")
	}
	return client.GetAlbum(ctx, id, opts...)
}

func (s *spotifyProvider) GetArtist(ctx context.Context, id spotify.ID, opts ...spotify.RequestOption) (*spotify.SimpleAlbumPage, error) {
	log := logger.From(ctx)
	log.Info("fetching spotify artist", "artist_id", id)
	albumTypes := []spotify.AlbumType{spotify.AlbumTypeAlbum, spotify.AlbumTypeSingle, spotify.AlbumTypeAppearsOn, spotify.AlbumTypeCompilation}
	client, err := s.withClient(ctx)
	if err != nil {
		log.Error("failed to acquire client")
	}
	return client.GetArtistAlbums(ctx, id, albumTypes, opts...)
}
