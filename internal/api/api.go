// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"audio-scraper/internal/adapters/itunes"
	"audio-scraper/internal/adapters/store"
	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/pool"
)

type Deps struct {
	Log      logger.Logger
	Metadata itunes.Provider
	Store    store.Provider
	Queue    *pool.DownloadWorkerPool
}

type Handlers struct {
	log      logger.Logger
	metadata itunes.Provider
	store    store.Provider
	queue    *pool.DownloadWorkerPool
}

func NewHandlers(deps *Deps) *Handlers {
	return &Handlers{log: deps.Log, metadata: deps.Metadata, store: deps.Store, queue: deps.Queue}
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

	var allChoices []store.Choice
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		log := log.With("query", query)
		results, err := h.metadata.Search(logger.Into(ctx, log), query)
		if err != nil {
			log.Error("metadata search failed", "error", err)
			http.Error(w, "metadata search failed", http.StatusInternalServerError)
			return
		}

		choices := processSearchData(results, log)
		allChoices = append(allChoices, choices...)
	}

	h.store.Set(requestID, allChoices)

	var labels []string
	for _, choice := range allChoices {
		labels = append(labels, choice.Label)
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
		log.Warn("invalid download request", "error", err)
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

	resolved := make([]*store.Choice, 0, len(req.Choices))
	for _, choice := range req.Choices {
		c := data.FindByLabel(choice)
		if c == nil {
			log.Warn("choice not found in stored data", "choice", choice)
			http.Error(w, "Choice not found: "+choice, http.StatusBadRequest)
			return
		}
		resolved = append(resolved, c)
	}

	reqID := req.RequestID
	go func(log logger.Logger, reqID string, choices []*store.Choice) {
		for _, c := range choices {
			clog := log.With("type", c.Type, "id", c.ID)
			clog.Info("processing choice")

			deps := addToQueueDeps{
				log:      clog,
				metadata: h.metadata,
				q:        h.queue,
			}

			switch c.Type {
			case constants.EntityTypeTrack:
				addTrackToQueue(deps, reqID, c.ID)
			case constants.EntityTypeAlbum:
				addAlbumToQueue(deps, reqID, c.ID)
			case constants.EntityTypeArtist:
				addArtistToQueue(deps, reqID, c.ID)
			default:
				clog.Warn("unknown choice type", "type", c.Type)
			}
		}
	}(log, reqID, resolved)

	w.WriteHeader(http.StatusAccepted)
}
