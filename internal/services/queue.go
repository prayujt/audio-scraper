// Package services implements a download worker pool for processing download jobs concurrently.
package services

import (
	"context"
	"sync"

	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

type DownloadWorkerPool struct {
	jobs    chan models.DownloadJob
	workers int

	log ports.Logger

	wg   sync.WaitGroup
	stop chan struct{}
}

func NewDownloadWorkerPool(
	workers int,
	log ports.Logger,
) *DownloadWorkerPool {
	p := &DownloadWorkerPool{
		jobs:    make(chan models.DownloadJob, 1000),
		workers: workers,
		log:     log.With("component", "DownloadWorkerPool"),
		stop:    make(chan struct{}),
	}

	p.start()
	return p
}

func (p *DownloadWorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *DownloadWorkerPool) worker(id int) {
	defer p.wg.Done()
	log := p.log.With("worker_id", id)

	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				log.Info("jobs channel closed, worker exiting")
				return
			}

			log := log.With("request_id", job.RequestID, "track_id", job.TrackID)

			log.Info("processing download job")
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
