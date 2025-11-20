package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"audio-scraper/internal/api"
	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/providers"
	"audio-scraper/internal/services"
)

func main() {
	log := logger.NewLogger()
	log.Debug("init starting")
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	log.Info("started server", "host", "0.0.0.0", "port", port)

	sp, err := providers.NewSpotifyProvider(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))
	if err != nil {
		log.Error("failed to initialize Spotify provider", "err", err)
		return
	}
	st := providers.NewStoreProvider(log)
	yt := providers.NewYTProvider()
	fs, err := providers.NewFSProvider(os.Getenv("MUSIC_HOME"))
	if err != nil {
		log.Error("failed to initialize filesystem provider", "err", err)
		return
	}

	poolSizeEnv := os.Getenv("WORKER_SIZE")
	poolSize, err := strconv.Atoi(poolSizeEnv)
	if err != nil || poolSize <= 0 {
		poolSize = constants.DownloadWorkerPoolSize
	}
	q := services.NewDownloadWorkerPool(poolSize, &services.Deps{
		Log: log,
		YT:  yt,
		FS:  fs,
	})
	h := api.NewHandlers(&api.Deps{
		Log:     log,
		Spotify: sp,
		Store:   st,
		Queue:   q,
	})
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
