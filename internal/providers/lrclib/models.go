package lrclib

import (
	"regexp"
	"strings"
)

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

type lrclibResponse struct {
	ID           int     `json:"id"`
	ArtistName   string  `json:"artistName"`
	TrackName    string  `json:"trackName"`
	AlbumName    string  `json:"albumName"`
	Duration     float32 `json:"duration"`
	SyncedLyrics string  `json:"syncedLyrics"`
	PlainLyrics  string  `json:"plainLyrics"`
}

func (r *lrclibResponse) ToLRC() *LRC {
	lrc := &LRC{
		Segments: nil,
		Full:     &r.PlainLyrics,
	}
	if strings.TrimSpace(r.SyncedLyrics) == "" {
		return lrc
	}

	re := regexp.MustCompile(`^\[(\d{2}:\d{2}(?:\.\d{1,3})?)\]\s*(.*)$`)

	lines := strings.Split(r.SyncedLyrics, "\n")
	segments := make([]LRCSegment, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		m := re.FindStringSubmatch(line)
		if len(m) != 3 {
			// line doesn't match [mm:ss.xx] pattern – skip it.
			continue
		}

		segments = append(segments, LRCSegment{
			Time: m[1],
			Text: m[2],
		})
	}
	if len(segments) == 0 {
		return lrc
	}

	lrc.Segments = segments
	return lrc
}
