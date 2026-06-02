// Package lrclibmock provides a test double for lrclib.Provider.
package lrclibmock

import (
	"context"

	"audio-scraper/internal/adapters/lrclib"
)

// Mock implements lrclib.Provider; set FindLyricsFunc to control behavior.
type Mock struct {
	FindLyricsFunc func(ctx context.Context, req *lrclib.LrclibRequest) (*lrclib.LRC, error)
}

var _ lrclib.Provider = (*Mock)(nil)

func (m *Mock) FindLyrics(ctx context.Context, req *lrclib.LrclibRequest) (*lrclib.LRC, error) {
	if m.FindLyricsFunc != nil {
		return m.FindLyricsFunc(ctx, req)
	}
	return nil, nil
}
