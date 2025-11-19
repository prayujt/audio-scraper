package api

import (
	"fmt"

	"github.com/zmb3/spotify/v2"

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
			Type: "track",
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
			Type: "album",
			ID:   a.ID.String(),
		}
		choices[label] = choice
	}

	for i := 0; i < min(artistCount, len(artists)); i++ {
		ar := artists[i]
		label := fmt.Sprintf("Artist: %s", ar.Name)

		choice := models.Choice{
			Type: "artist",
			ID:   ar.ID.String(),
		}
		choices[label] = choice
	}

	return choices, nil
}
