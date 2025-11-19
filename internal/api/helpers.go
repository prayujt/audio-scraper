package api

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
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

func addTrackToQueue(deps addToQueueDeps, requestID string, trackID spotify.ID) {
	ctx := context.Background()
	log := deps.log.With("track_id", trackID)
	log.Info("adding track to download queue")

	track, err := deps.sp.GetTrack(logger.Into(ctx, log), spotify.ID(trackID))
	if err != nil {
		log.Error("failed to fetch track details", "err", err)
		return
	}
	err = deps.q.Enqueue(ctx, models.DownloadJob{
		RequestID: requestID,
		TrackID:   trackID.String(),
		Track:     track.Name,
		Album:     track.Album.Name,
		Artist:    track.Artists[0].Name,
	})
	if err != nil {
		log.Error("failed to add track to download queue", "err", err)
		return
	}

	log.Info("track added to download queue successfully")
}

func addAlbumToQueue(deps addToQueueDeps, requestID string, albumID spotify.ID) {
	ctx := context.Background()
	log := deps.log.With("album_id", albumID)

	album, err := deps.sp.GetAlbum(logger.Into(ctx, log), albumID)
	if err != nil {
		log.Error("failed to fetch album details", "err", err)
		return
	}

	for _, track := range album.Tracks.Tracks {
		addTrackToQueue(deps, requestID, track.ID)
	}
	log.Info("album added to download queue successfully")
}

func addArtistToQueue(deps addToQueueDeps, requestID string, artistID spotify.ID) {
	ctx := context.Background()
	log := deps.log.With("artist_id", artistID)

	artist, err := deps.sp.GetArtist(logger.Into(ctx, log), artistID)
	if err != nil {
		log.Error("failed to fetch artist details", "err", err)
		return
	}

	for _, album := range artist.Albums {
		addAlbumToQueue(deps, requestID, album.ID)
	}
	log.Info("artist added to download queue successfully")
}
