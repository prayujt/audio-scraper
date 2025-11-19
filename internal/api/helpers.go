package api

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/constants"
	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

func processSearchData(result *spotify.SearchResult, log ports.Logger) (map[string]models.Choice, error) {
	trackCount := 10
	albumCount := 5
	artistCount := 3

	var tracks []spotify.FullTrack
	var albums []spotify.SimpleAlbum
	var artists []spotify.FullArtist
	if result.Tracks != nil {
		tracks = result.Tracks.Tracks
		log.Debug("tracks found", "count", len(tracks))
	}
	if result.Albums != nil {
		albums = result.Albums.Albums
		log.Debug("albums found", "count", len(albums))
	}
	if result.Artists != nil {
		artists = result.Artists.Artists
		log.Debug("artists found", "count", len(artists))
	}

	if len(albums) < albumCount {
		trackCount += albumCount - len(albums)
		albumCount = len(albums)
	}
	if len(artists) < artistCount {
		trackCount += artistCount - len(artists)
		artistCount = len(artists)
	}
	log.Debug("reallocated counts", "tracks", trackCount, "albums", albumCount, "artists", artistCount)

	choices := make(map[string]models.Choice)
	for i := 0; i < min(trackCount, len(tracks)); i++ {
		t := tracks[i]
		artistName := ""
		if len(t.Artists) > 0 {
			artistName = t.Artists[0].Name
		}
		label := fmt.Sprintf("Track: %s - %s [%s]", t.Name, artistName, t.Album.Name)

		choice := models.Choice{
			Type: constants.SpotifyEntityTypeTrack,
			ID:   t.ID.String(),
		}
		choices[label] = choice
	}

	for i := 0; i < min(albumCount, len(albums)); i++ {
		a := albums[i]
		artistName := ""
		if len(a.Artists) > 0 {
			artistName = a.Artists[0].Name
		}
		label := fmt.Sprintf("Album: %s - %s", a.Name, artistName)

		choice := models.Choice{
			Type: constants.SpotifyEntityTypeAlbum,
			ID:   a.ID.String(),
		}
		choices[label] = choice
	}

	for i := 0; i < min(artistCount, len(artists)); i++ {
		ar := artists[i]
		label := fmt.Sprintf("Artist: %s", ar.Name)

		choice := models.Choice{
			Type: constants.SpotifyEntityTypeArtist,
			ID:   ar.ID.String(),
		}
		choices[label] = choice
	}

	return choices, nil
}

type addToQueueDeps struct {
	log ports.Logger
	sp  ports.SpotifyProvider
	q   ports.DownloadQueue
}

func addTrackToQueue(deps addToQueueDeps, requestID string, trackID string) {
	ctx := context.Background()
	log := deps.log.With("track_id", trackID)
	log.Info("adding track to download queue")

	err := deps.q.Enqueue(ctx, models.DownloadJob{
		RequestID: requestID,
		TrackID:   trackID,
		Track:     "",
		Album:     "",
		Artist:    "",
	})
	if err != nil {
		log.Error("failed to add track to download queue", "err", err)
		return
	}

	log.Info("track added to download queue successfully")
}

func addAlbumToQueue(deps addToQueueDeps, requestID string, albumID string) {
	ctx := context.Background()
	log := deps.log.With("album_id", albumID)

	err := deps.q.Enqueue(ctx, models.DownloadJob{
		RequestID: requestID,
		TrackID:   "",
		Track:     "",
		Album:     "",
		Artist:    "",
	})
	if err != nil {
		log.Error("failed to add track to download queue", "err", err)
		return
	}

	log.Info("album added to download queue successfully")
}

func addArtistToQueue(deps addToQueueDeps, requestID string, artistID string) {
	ctx := context.Background()
	log := deps.log.With("artist_id", artistID)

	err := deps.q.Enqueue(ctx, models.DownloadJob{
		RequestID: requestID,
		TrackID:   "",
		Track:     "",
		Album:     "",
		Artist:    "",
	})
	if err != nil {
		log.Error("failed to add track to download queue", "err", err)
		return
	}

	log.Info("artist added to download queue successfully")
}
