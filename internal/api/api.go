// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/logger"
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
	query := r.URL.Query().Get("q")
	if query == "" {
		log.Warn("search query parameter 'q' is missing")
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}
	log = h.log.With("query", query)
	results, err := h.spotify.Search(logger.Into(ctx, log), query, spotify.SearchTypePlaylist|spotify.SearchTypeAlbum)
	if err != nil {
		log.Error("spotify search failed", "err", err)
		http.Error(w, "spotify search failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Download endpoint not implemented"))
}
