// Package filesystem defines the local library port: where downloaded audio is
// placed on disk and how it is tagged. The concrete implementation lives in the
// impl subpackage.
package filesystem

import (
	"context"

	"audio-scraper/internal/models"
)

// Provider manages on-disk placement and tagging of downloaded tracks.
type Provider interface {
	// InitializePath creates the destination directory tree and returns the
	// output file path for the job.
	InitializePath(ctx context.Context, job *models.DownloadJob) (string, error)
	// TagFile writes ID3 metadata, cover art and lyrics to the file at filePath.
	TagFile(ctx context.Context, filePath string, job *models.DownloadJob) error
}
