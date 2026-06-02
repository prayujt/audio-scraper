// Command audio-scraper is the HTTP server entrypoint: it loads configuration,
// wires up the adapters, and serves the search/download API.
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	filesystemimpl "audio-scraper/internal/adapters/filesystem/impl"
	itunesimpl "audio-scraper/internal/adapters/itunes/impl"
	lrclibimpl "audio-scraper/internal/adapters/lrclib/impl"
	storeimpl "audio-scraper/internal/adapters/store/impl"
	subsonicimpl "audio-scraper/internal/adapters/subsonic/impl"
	youtubeimpl "audio-scraper/internal/adapters/youtube/impl"
	"audio-scraper/internal/api"
	"audio-scraper/internal/config"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/pool"
)

func main() {
	log := logger.NewLogger()
	log.Debug("init starting")

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		return
	}
	log.Info("started server", "host", "0.0.0.0", "port", cfg.APIPort)

	md := itunesimpl.New()
	st := storeimpl.New(log)
	yt := youtubeimpl.New()
	lrc := lrclibimpl.New()
	ss := subsonicimpl.New(cfg.SubsonicURL, cfg.SubsonicUser, cfg.SubsonicPassword)
	// Verify Subsonic credentials up front. When no URL is configured this
	// warns and returns nil; when one is, a failure is fatal.
	if err := ss.Ping(logger.Into(context.Background(), log)); err != nil {
		log.Error("subsonic ping failed", "error", err)
		return
	}
	fs, err := filesystemimpl.New(cfg.MusicHome, lrc)
	if err != nil {
		log.Error("failed to initialize filesystem provider", "error", err)
		return
	}

	q := pool.NewDownloadWorkerPool(cfg.WorkerSize, &pool.Deps{
		Log:      log,
		YT:       yt,
		FS:       fs,
		Subsonic: ss,
	})
	h := api.NewHandlers(&api.Deps{
		Log:      log,
		Metadata: md,
		Store:    st,
		Queue:    q,
	})
	router := mux.NewRouter()
	router.HandleFunc("/", h.HealthHandler).Methods("GET")
	router.HandleFunc("/search", h.Search).Methods("GET")
	router.HandleFunc("/download", h.Download).Methods("POST")

	server := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("0.0.0.0:%s", cfg.APIPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Error("server failed", "error", err)
	}
}
