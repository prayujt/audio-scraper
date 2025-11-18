package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"audio-scraper/internal/api"
	"audio-scraper/internal/logger"
)

func main() {
	log := logger.NewLogger()
	log.Debug("init starting")
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	log.Info("started server", "host", "0.0.0.0", "port", port)

	h := api.NewHandlers(log)
	router := mux.NewRouter()
	router.HandleFunc("/", h.HealthHandler).Methods("GET")
	router.HandleFunc("/search", h.Search).Methods("GET")
	router.HandleFunc("/download", h.Download).Methods("POST")

	server := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Error("server failed", "err", err)
	}
}
