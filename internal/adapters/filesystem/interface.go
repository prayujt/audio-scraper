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
	// ReplaceAudio swaps the audio of the existing file for job (located by
	// Artist/Album/Track) with freshly downloaded audio, preserving the file's
	// existing tags. download must write an mp3 to the dest path it is given.
	// The original file is left intact if anything fails.
	ReplaceAudio(ctx context.Context, job *models.DownloadJob, download func(ctx context.Context, dest string) error) error
}
