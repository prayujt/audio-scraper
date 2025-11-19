// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"encoding/json"
	"maps"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/zmb3/spotify/v2"

	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

type Deps struct {
	Log     ports.Logger
	Spotify ports.SpotifyProvider
	Store   ports.StoreProvider
	Queue   ports.DownloadQueue
}

type Handlers struct {
	log     ports.Logger
	spotify ports.SpotifyProvider
	store   ports.StoreProvider
	queue   ports.DownloadQueue
}

func NewHandlers(deps *Deps) *Handlers {
	return &Handlers{log: deps.Log, spotify: deps.Spotify, store: deps.Store, queue: deps.Queue}
}

func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Info("received request to health check endpoint")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := uuid.New().String()
	log := h.log.With("handler", "Search", "request_id", requestID)
	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		log.Warn("search query parameter 'q' is missing")
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}
	queries := strings.Split(searchQuery, ",")

	allChoices := make(map[string]models.Choice)
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

		maps.Copy(allChoices, choices)
	}

	h.store.Set(requestID, allChoices)

	var labels []string
	for label := range allChoices {
		labels = append(labels, label)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SearchResponse{
		RequestID: requestID,
		Choices:   labels,
	})
}

func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	log := h.log.With("handler", "Download")

	var req models.DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid download request", "err", err)
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	log = log.With("request_id", req.RequestID)

	data, found := h.store.Get(req.RequestID)
	if !found {
		log.Warn("request ID not found in store")
		http.Error(w, "Request ID not found", http.StatusBadRequest)
		return
	}

	log.Info("download request received", "selections", req.Choices)
	for _, choice := range req.Choices {
		log := log.With("choice", choice)
		c, exists := data[choice]
		if !exists {
			log.Warn("choice not found in stored data")
			http.Error(w, "Choice not found: "+choice, http.StatusBadRequest)
			return
		}
		log.Info("processing choice", "type", c.Type, "id", c.ID)

		deps := addToQueueDeps{
			log: log,
			sp:  h.spotify,
			q:   h.queue,
		}
		switch c.Type {
		case constants.SpotifyEntityTypeTrack:
			addTrackToQueue(deps, req.RequestID, c.ID)
		case constants.SpotifyEntityTypeAlbum:
			addAlbumToQueue(deps, req.RequestID, c.ID)
		case constants.SpotifyEntityTypeArtist:
			addArtistToQueue(deps, req.RequestID, c.ID)
		}
	}

	w.WriteHeader(http.StatusOK)
}
