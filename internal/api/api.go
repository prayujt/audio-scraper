// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

type Deps struct {
	Log     ports.Logger
	Spotify ports.SpotifyProvider
}

type Handlers struct {
	log     ports.Logger
	spotify ports.SpotifyProvider
}

func NewHandlers(deps *Deps) *Handlers {
	return &Handlers{log: deps.Log, spotify: deps.Spotify}
}

func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Info("received request to health check endpoint")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("handler", "Search")
	queryParam := r.URL.Query().Get("q")
	if queryParam == "" {
		log.Warn("search query parameter 'q' is missing")
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	queries := strings.Split(queryParam, ",")
	var allResults []*spotify.SearchResult

	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		queryLog := log.With("query", query)
		results, err := h.spotify.Search(logger.Into(ctx, queryLog), query, spotify.SearchTypeArtist|spotify.SearchTypeAlbum|spotify.SearchTypeTrack)
		if err != nil {
			queryLog.Error("spotify search failed", "err", err)
			http.Error(w, "spotify search failed", http.StatusInternalServerError)
			return
		}

		allResults = append(allResults, results)
	}

	var allChoices [][]models.Choice
	for _, result := range allResults {
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
			log.Debug("reallocating unused album slots to tracks", "count", albumCount-len(albums))
			trackCount += albumCount - len(albums)
			albumCount = len(albums)
		}
		if len(artists) < artistCount {
			log.Debug("reallocating unused artist slots to tracks", "count", artistCount-len(artists))
			trackCount += artistCount - len(artists)
			artistCount = len(artists)
		}
		log.Debug("final counts", "tracks", trackCount, "albums", albumCount, "artists", artistCount)

		var choices []models.Choice
		for i := 0; i < min(trackCount, len(tracks)); i++ {
			t := tracks[i]
			artistName := ""
			if len(t.Artists) > 0 {
				artistName = t.Artists[0].Name
			}
			label := fmt.Sprintf("Track: %s - %s [%s]", t.Name, artistName, t.Album.Name)

			choices = append(choices, models.Choice{
				Type:  "track",
				ID:    t.ID.String(),
				Label: label,
			})
		}

		for i := 0; i < min(albumCount, len(albums)); i++ {
			a := albums[i]
			artistName := ""
			if len(a.Artists) > 0 {
				artistName = a.Artists[0].Name
			}
			label := fmt.Sprintf("Album: %s - %s", a.Name, artistName)

			choices = append(choices, models.Choice{
				Type:  "album",
				ID:    a.ID.String(),
				Label: label,
			})
		}

		for i := 0; i < min(artistCount, len(artists)); i++ {
			ar := artists[i]
			label := fmt.Sprintf("Artist: %s", ar.Name)

			choices = append(choices, models.Choice{
				Type:  "artist",
				ID:    ar.ID.String(),
				Label: label,
			})
		}
		allChoices = append(allChoices, choices)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allChoices)
}

func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Download endpoint not implemented"))
}
