// Package api implements HTTP handlers for the audio scraper service.
package api

import (
	"net/http"

	"audio-scraper/internal/ports"
)

type Deps struct {
	Log ports.Logger
	Sp  ports.SpotifyProvider
}

type Handlers struct {
	log ports.Logger
}

func NewHandlers(deps *Deps) *Handlers {
	return &Handlers{log: deps.Log}
}

func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Info("received request to health check endpoint")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Search endpoint not implemented"))
}

func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Download endpoint not implemented"))
}
