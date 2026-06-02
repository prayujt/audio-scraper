// Package filesystemmock provides a test double for filesystem.Provider.
package filesystemmock

import (
	"context"

	"audio-scraper/internal/adapters/filesystem"
	"audio-scraper/internal/models"
)

// Mock implements filesystem.Provider; set the *Func fields to control behavior.
type Mock struct {
	InitializePathFunc func(ctx context.Context, job *models.DownloadJob) (string, error)
	TagFileFunc        func(ctx context.Context, filePath string, job *models.DownloadJob) error
}

var _ filesystem.Provider = (*Mock)(nil)

func (m *Mock) InitializePath(ctx context.Context, job *models.DownloadJob) (string, error) {
	if m.InitializePathFunc != nil {
		return m.InitializePathFunc(ctx, job)
	}
	return "", nil
}

func (m *Mock) TagFile(ctx context.Context, filePath string, job *models.DownloadJob) error {
	if m.TagFileFunc != nil {
		return m.TagFileFunc(ctx, filePath, job)
	}
	return nil
}
