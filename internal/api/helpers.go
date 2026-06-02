package api

import (
	"context"
	"fmt"

	"audio-scraper/internal/adapters/itunes"
	"audio-scraper/internal/adapters/store"
	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/pool"
)

func processSearchData(result models.SearchResult, log logger.Logger) []store.Choice {
	trackCount := 10
	albumCount := 5
	artistCount := 3

	tracks := result.Tracks
	albums := result.Albums
	artists := result.Artists
	log.Debug("search results", "tracks", len(tracks), "albums", len(albums), "artists", len(artists))

	if len(albums) < albumCount {
		trackCount += albumCount - len(albums)
		albumCount = len(albums)
	}
	if len(artists) < artistCount {
		trackCount += artistCount - len(artists)
		artistCount = len(artists)
	}
	log.Debug("reallocated counts", "tracks", trackCount, "albums", albumCount, "artists", artistCount)

	var choices []store.Choice
	for i := 0; i < min(trackCount, len(tracks)); i++ {
		t := tracks[i]
		label := fmt.Sprintf("Track: %s - %s [%s]", t.Name, t.Artist, t.Album)
		choices = append(choices, store.Choice{
			Type:  constants.EntityTypeTrack,
			ID:    t.ID,
			Label: label,
		})
	}

	for i := 0; i < min(albumCount, len(albums)); i++ {
		a := albums[i]
		label := fmt.Sprintf("Album: %s - %s", a.Name, a.Artist)
		choices = append(choices, store.Choice{
			Type:  constants.EntityTypeAlbum,
			ID:    a.ID,
			Label: label,
		})
	}

	for i := 0; i < min(artistCount, len(artists)); i++ {
		ar := artists[i]
		label := fmt.Sprintf("Artist: %s", ar.Name)
		choices = append(choices, store.Choice{
			Type:  constants.EntityTypeArtist,
			ID:    ar.ID,
			Label: label,
		})
	}

	return choices
}

type addToQueueDeps struct {
	log      logger.Logger
	metadata itunes.Provider
	q        *pool.DownloadWorkerPool
}

func trackToJob(requestID string, t models.Track) models.DownloadJob {
	return models.DownloadJob{
		RequestID:    requestID,
		TrackID:      t.ID,
		Track:        t.Name,
		Album:        t.Album,
		Artist:       t.Artist,
		ReleaseDate:  t.ReleaseDate,
		TrackNumber:  t.TrackNumber,
		Duration:     t.Duration,
		ThumbnailURL: t.ArtworkURL,
	}
}

func addTrackToQueue(deps addToQueueDeps, requestID string, trackID string) {
	ctx := context.Background()
	log := deps.log.With("track_id", trackID)
	log.Info("adding track to download queue")

	track, err := deps.metadata.GetTrack(logger.Into(ctx, log), trackID)
	if err != nil {
		log.Error("failed to fetch track details", "error", err)
		return
	}
	if err := deps.q.Enqueue(ctx, trackToJob(requestID, track)); err != nil {
		log.Error("failed to add track to download queue", "error", err)
		return
	}

	log.Info("track added to download queue successfully")
}

func addAlbumToQueue(deps addToQueueDeps, requestID string, albumID string) {
	ctx := context.Background()
	log := deps.log.With("album_id", albumID)

	album, err := deps.metadata.GetAlbum(logger.Into(ctx, log), albumID)
	if err != nil {
		log.Error("failed to fetch album details", "error", err)
		return
	}

	for _, track := range album.Tracks {
		if err := deps.q.Enqueue(ctx, trackToJob(requestID, track)); err != nil {
			log.Error("failed to add album track to download queue", "error", err, "track_id", track.ID)
		}
	}
	log.Info("album added to download queue successfully")
}

func addArtistToQueue(deps addToQueueDeps, requestID string, artistID string) {
	ctx := context.Background()
	log := deps.log.With("artist_id", artistID)

	artist, err := deps.metadata.GetArtist(logger.Into(ctx, log), artistID)
	if err != nil {
		log.Error("failed to fetch artist details", "error", err)
		return
	}

	for _, album := range artist.Albums {
		addAlbumToQueue(deps, requestID, album.ID)
	}
	log.Info("artist added to download queue successfully")
}
