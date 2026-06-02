// Package youtubemock provides a test double for youtube.Provider.
package youtubemock

import (
	"context"

	"audio-scraper/internal/adapters/youtube"
)

// Mock implements youtube.Provider; set the *Func fields to control behavior.
type Mock struct {
	SearchFunc     func(ctx context.Context, track, artist string, duration int) (string, error)
	CandidatesFunc func(ctx context.Context, track, artist string, duration int) ([]youtube.Candidate, error)
	DownloadFunc   func(ctx context.Context, path, videoURL string) (int, error)
}

var _ youtube.Provider = (*Mock)(nil)

func (m *Mock) Search(ctx context.Context, track, artist string, duration int) (string, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, track, artist, duration)
	}
	return "", nil
}

func (m *Mock) Candidates(ctx context.Context, track, artist string, duration int) ([]youtube.Candidate, error) {
	if m.CandidatesFunc != nil {
		return m.CandidatesFunc(ctx, track, artist, duration)
	}
	return nil, nil
}

func (m *Mock) Download(ctx context.Context, path, videoURL string) (int, error) {
	if m.DownloadFunc != nil {
		return m.DownloadFunc(ctx, path, videoURL)
	}
	return -1, nil
}
