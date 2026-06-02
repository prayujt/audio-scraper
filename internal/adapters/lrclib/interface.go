// Package lrclib defines the lyrics provider port and its contract types. The
// concrete implementation lives in the impl subpackage.
package lrclib

import (
	"context"
	"strings"
)

// Provider fetches lyrics for a track.
type Provider interface {
	FindLyrics(ctx context.Context, req *LrclibRequest) (*LRC, error)
}

type LrclibRequest struct {
	Artist   string
	Album    string
	Track    string
	Duration int
}

type LRCSegment struct {
	Time string
	Text string
}

type LRC struct {
	Segments []LRCSegment
	Full     *string
}

func (l *LRC) SyncedToText() string {
	var b strings.Builder
	for _, s := range l.Segments {
		// Each line: [mm:ss.xx] text
		b.WriteString("[")
		b.WriteString(s.Time)
		b.WriteString("] ")
		b.WriteString(s.Text)
		b.WriteRune('\n')
	}
	return b.String()
}
