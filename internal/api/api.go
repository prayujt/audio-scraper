package api

import (
	"net/http"

	"audio-scraper/internal/ports"
)

type handlers struct {
	log ports.Logger
}

func NewHandlers(l ports.Logger) ports.Handlers {
	return &handlers{log: l}
}

func (h *handlers) Log() ports.Logger { return h.log }

func (h *handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *handlers) Search(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Search endpoint not implemented"))
}

func (h *handlers) Download(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Download endpoint not implemented"))
}
