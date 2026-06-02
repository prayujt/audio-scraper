// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"audio-scraper/internal/adapters/itunes"
	"audio-scraper/internal/adapters/store"
	"audio-scraper/internal/adapters/subsonic"
	"audio-scraper/internal/adapters/youtube"
	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/pool"
)

type Deps struct {
	Log      logger.Logger
	Metadata itunes.Provider
	Subsonic subsonic.Provider
	YouTube  youtube.Provider
	Store    store.Provider
	Queue    *pool.DownloadWorkerPool
}

type Handlers struct {
	log      logger.Logger
	metadata itunes.Provider
	subsonic subsonic.Provider
	youtube  youtube.Provider
	store    store.Provider
	queue    *pool.DownloadWorkerPool
}

func NewHandlers(deps *Deps) *Handlers {
	return &Handlers{
		log:      deps.Log,
		metadata: deps.Metadata,
		subsonic: deps.Subsonic,
		youtube:  deps.YouTube,
		store:    deps.Store,
		queue:    deps.Queue,
	}
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

// LibrarySearch performs a partial search against the Subsonic library and
// returns the matching songs as selectable choices (replacement flow, step 1).
func (h *Handlers) LibrarySearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := uuid.New().String()
	log := h.log.With("handler", "LibrarySearch", "request_id", requestID)

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		log.Warn("search query parameter 'q' is missing")
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	res, err := h.subsonic.Search(logger.Into(ctx, log.With("query", query)), query)
	if err != nil {
		log.Error("subsonic search failed", "error", err)
		http.Error(w, "subsonic search failed", http.StatusInternalServerError)
		return
	}

	choices := songsToChoices(res.Songs)
	h.store.Set(requestID, choices)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SearchResponse{
		RequestID: requestID,
		Choices:   choiceLabels(choices),
	})
}

// LibraryCandidates returns the ranked YouTube candidates for a previously
// selected library song (replacement flow, step 2).
func (h *Handlers) LibraryCandidates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("handler", "LibraryCandidates")

	var req models.ChoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid candidates request", "error", err)
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	log = log.With("request_id", req.RequestID)

	data, found := h.store.Get(req.RequestID)
	if !found {
		http.Error(w, "Request ID not found", http.StatusBadRequest)
		return
	}
	song := data.FindByLabel(req.Choice)
	if song == nil {
		http.Error(w, "Choice not found: "+req.Choice, http.StatusBadRequest)
		return
	}

	cands, err := h.youtube.Candidates(logger.Into(ctx, log), song.Track, song.Artist, song.Duration)
	if err != nil {
		log.Error("youtube candidate search failed", "error", err)
		http.Error(w, "youtube candidate search failed", http.StatusInternalServerError)
		return
	}

	choices := candidatesToChoices(cands, song)
	h.store.Set(req.RequestID, choices)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SearchResponse{
		RequestID: req.RequestID,
		Choices:   choiceLabels(choices),
	})
}

// Replace enqueues a replacement job for a chosen YouTube candidate
// (replacement flow, step 3).
func (h *Handlers) Replace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("handler", "Replace")

	var req models.ChoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid replace request", "error", err)
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	log = log.With("request_id", req.RequestID)

	data, found := h.store.Get(req.RequestID)
	if !found {
		http.Error(w, "Request ID not found", http.StatusBadRequest)
		return
	}
	c := data.FindByLabel(req.Choice)
	if c == nil {
		http.Error(w, "Choice not found: "+req.Choice, http.StatusBadRequest)
		return
	}

	job := models.DownloadJob{
		RequestID:  req.RequestID,
		Track:      c.Track,
		Album:      c.Album,
		Artist:     c.Artist,
		Duration:   c.Duration,
		YouTubeURL: c.URL,
	}
	if err := h.queue.Enqueue(ctx, job); err != nil {
		log.Error("failed to enqueue replacement", "error", err)
		http.Error(w, "failed to enqueue replacement", http.StatusInternalServerError)
		return
	}

	log.Info("replacement queued", "track", c.Track, "url", c.URL)
	w.WriteHeader(http.StatusAccepted)
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
