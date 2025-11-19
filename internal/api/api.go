// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"encoding/json"
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
	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		log.Warn("search query parameter 'q' is missing")
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}
	queries := strings.Split(searchQuery, ",")

	var totalChoices [][]models.Choice
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		log := log.With("query", query)
		results, err := h.spotify.Search(logger.Into(ctx, log), query, spotify.SearchTypeArtist|spotify.SearchTypeAlbum|spotify.SearchTypeTrack)
		if err != nil {
			log.Error("spotify search failed", "err", err)
			http.Error(w, "spotify search failed", http.StatusInternalServerError)
			return
		}

		choices, err := processSearchData(results, log)
		if err != nil {
			log.Error("processing search data failed", "err", err)
			http.Error(w, "processing search data failed", http.StatusInternalServerError)
			return
		}

		totalChoices = append(totalChoices, choices)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(totalChoices)
}

func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Download endpoint not implemented"))
}
