// Package pool implements a download worker pool for processing download jobs concurrently.
package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"audio-scraper/internal/adapters/filesystem"
	"audio-scraper/internal/adapters/subsonic"
	"audio-scraper/internal/adapters/youtube"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
)

// scanInterval is how often the pool coalesces completed downloads into a
// single Subsonic rescan, rather than scanning after every track.
const scanInterval = 30 * time.Second

type DownloadWorkerPool struct {
	jobs    chan models.DownloadJob
	workers int

	log      logger.Logger
	yt       youtube.Provider
	fs       filesystem.Provider
	subsonic subsonic.Provider

	// dirty is set by workers when a download completes and cleared by the
	// scanner when it triggers a rescan, batching bursts into one scan.
	dirty atomic.Bool

	wg   sync.WaitGroup
	stop chan struct{}
}

type Deps struct {
	Log      logger.Logger
	YT       youtube.Provider
	FS       filesystem.Provider
	Subsonic subsonic.Provider
}

func NewDownloadWorkerPool(
	workers int,
	deps *Deps,
) *DownloadWorkerPool {
	p := &DownloadWorkerPool{
		jobs:    make(chan models.DownloadJob, 1000),
		workers: workers,
		log:      deps.Log.With("component", "DownloadWorkerPool"),
		yt:       deps.YT,
		fs:       deps.FS,
		subsonic: deps.Subsonic,
		stop:     make(chan struct{}),
	}

	p.start()
	return p
}

func (p *DownloadWorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.wg.Add(1)
	go p.scanner()
}

// scanner periodically triggers a Subsonic rescan if any downloads have
// completed since the last scan, coalescing bursts into a single call. It
// flushes one final time on shutdown.
func (p *DownloadWorkerPool) scanner() {
	defer p.wg.Done()
	log := p.log.With("component", "subsonic-scanner")
	ctx := context.Background()

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.flushScan(ctx, log)
		case <-p.stop:
			p.flushScan(ctx, log)
			return
		}
	}
}

// flushScan triggers a rescan when the library is dirty and clears the flag.
// On failure it re-marks dirty so the next tick retries.
func (p *DownloadWorkerPool) flushScan(ctx context.Context, log logger.Logger) {
	if !p.dirty.Swap(false) {
		return
	}
	log.Info("flushing batched subsonic rescan")
	if err := p.subsonic.StartScan(logger.Into(ctx, log)); err != nil {
		log.Error("subsonic rescan failed", "error", err)
		p.dirty.Store(true)
	}
}

func (p *DownloadWorkerPool) worker(id int) {
	defer p.wg.Done()
	log := p.log.With("worker_id", id)
	ctx := context.Background()

	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				log.Info("jobs channel closed, worker exiting")
				return
			}

			log := log.With("request_id", job.RequestID, "track_id", job.TrackID)

			// Replacement job: swap audio for an existing file, preserving tags.
			if job.YouTubeURL != "" {
				rlog := log.With("youtube_url", job.YouTubeURL, "track", job.Track)
				rlog.Info("processing replacement job")
				j := job
				err := p.fs.ReplaceAudio(logger.Into(ctx, rlog), &j, func(ctx context.Context, dest string) error {
					_, derr := p.yt.Download(ctx, dest, j.YouTubeURL)
					return derr
				})
				if err != nil {
					rlog.Error("replacement failed", "error", err)
					continue
				}
				p.dirty.Store(true)
				rlog.Info("replacement job completed successfully")
				continue
			}

			log.Info("processing download job")

			videoURL, err := p.yt.Search(logger.Into(ctx, log), job.Track, job.Artist, job.Duration)
			if err != nil {
				log.Error("yt search failed", "error", err)
				continue
			}
			log = log.With("video_url", videoURL)

			path, err := p.fs.InitializePath(logger.Into(ctx, log), &job)
			if err != nil {
				log.Error("failed to initialize filesystem path", "error", err)
				continue
			}

			duration, err := p.yt.Download(logger.Into(ctx, log), path, videoURL)
			if err != nil {
				log.Error("yt download failed", "error", err)
				continue
			}
			// use more accurate duration from video
			if duration != -1 {
				job.Duration = duration
			}

			err = p.fs.TagFile(logger.Into(ctx, log), path, &job)
			if err != nil {
				log.Error("failed to tag file", "error", err)
				continue
			}

			// Mark the library dirty so the scanner batches a rescan, rather
			// than scanning after every single track.
			p.dirty.Store(true)

			log.Info("download job completed successfully")
		case <-p.stop:
			log.Info("received stop signal, worker exiting")
			return
		}
	}
}

func (p *DownloadWorkerPool) Enqueue(ctx context.Context, job models.DownloadJob) error {
	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *DownloadWorkerPool) Shutdown() {
	close(p.stop)
	close(p.jobs)
	p.wg.Wait()
}
