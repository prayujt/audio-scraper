// Package youtubeimpl implements the youtube.Provider port using yt-dlp for
// both searching (ytsearch) and downloading.
package youtubeimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"

	"github.com/faiface/beep/mp3"

	"audio-scraper/internal/adapters/youtube"
	"audio-scraper/internal/logger"
)

// searchResults is how many candidates ytsearch returns for ranking.
const searchResults = 8

// Client is the YouTube-backed audio provider.
type Client struct{}

var _ youtube.Provider = (*Client)(nil)

// New returns a YouTube audio provider.
func New() *Client {
	return &Client{}
}

// ytEntry is a single ytsearch result (flat-playlist mode).
type ytEntry struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Uploader string  `json:"uploader"`
	Channel  string  `json:"channel"`
	Duration float64 `json:"duration"`
}

type ytSearchResponse struct {
	Entries []ytEntry `json:"entries"`
}

func (y *Client) Search(ctx context.Context, track, artist string, duration int) (string, error) {
	log := logger.From(ctx)
	query := strings.TrimSpace(track + " " + artist)
	log.Info("performing yt search", "query", query, "duration", duration)

	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		"--no-warnings",
		"--flat-playlist",
		"-J",
		fmt.Sprintf("ytsearch%d:%s", searchResults, query),
	)
	out, err := cmd.Output()
	if err != nil {
		log.Error("yt search command failed", "error", err)
		return "", errors.New("yt search failed")
	}

	var res ytSearchResponse
	if err := json.Unmarshal(out, &res); err != nil {
		log.Error("failed to parse yt search output", "error", err)
		return "", errors.New("yt search failed")
	}
	if len(res.Entries) == 0 {
		return "", errors.New("yt search returned no results")
	}

	best := -1
	bestScore := math.Inf(-1)
	for i, e := range res.Entries {
		if e.ID == "" {
			continue
		}
		if s := score(track, artist, duration, e); s > bestScore {
			bestScore = s
			best = i
		}
	}
	if best == -1 {
		return "", errors.New("yt search returned no usable results")
	}

	selected := res.Entries[best]
	url := "https://www.youtube.com/watch?v=" + selected.ID
	log.Info("yt search selected", "url", url, "title", selected.Title, "score", bestScore)
	return url, nil
}

// score rates how well a search result matches the desired track. Higher is
// better. It combines title containment, artist presence and duration
// closeness, mirroring the intent of the previous similarity matcher.
func score(track, artist string, duration int, e ytEntry) float64 {
	title := normalize(e.Title)
	haystack := normalize(e.Title + " " + e.Uploader + " " + e.Channel)
	wantTrack := normalize(track)
	wantArtist := normalize(artist)

	var s float64

	// Title match: exact containment is a strong signal, otherwise fall back to
	// token overlap.
	if wantTrack != "" && strings.Contains(title, wantTrack) {
		s += 2.0
	} else {
		s += jaccard(strings.Fields(title), strings.Fields(wantTrack))
	}

	// Artist match: fraction of artist tokens present in title/uploader/channel.
	artistTokens := strings.Fields(wantArtist)
	if len(artistTokens) > 0 {
		hits := 0
		for _, tok := range artistTokens {
			if strings.Contains(haystack, tok) {
				hits++
			}
		}
		s += float64(hits) / float64(len(artistTokens))
	}

	// Duration match: heavily reward results close to the expected length, which
	// filters out extended mixes, loops and snippets.
	if duration > 0 && e.Duration > 0 {
		ds := 1 - math.Abs(e.Duration-float64(duration))/float64(duration)
		if ds < 0 {
			ds = 0
		}
		s += 2.0 * ds
	}

	return s
}

// normalize lowercases s and reduces it to space-separated alphanumeric tokens.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// jaccard returns the Jaccard similarity of two token slices (0..1).
func jaccard(a, b []string) float64 {
	setA := make(map[string]struct{}, len(a))
	for _, x := range a {
		setA[x] = struct{}{}
	}
	setB := make(map[string]struct{}, len(b))
	for _, x := range b {
		setB[x] = struct{}{}
	}
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}

	inter := 0
	for x := range setA {
		if _, ok := setB[x]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	return float64(inter) / float64(union)
}

func (y *Client) Download(ctx context.Context, path, videoURL string) (int, error) {
	log := logger.From(ctx)
	log.Info("starting yt-dlp download", "path", path)
	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		"-q",
		"-x",
		"--audio-quality", "0",
		"--audio-format", "mp3",
		"-o", path,
		videoURL,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("yt-dlp command failed", "error", err, "output", string(output))
		return -1, errors.New("yt-dlp download failed")
	}

	duration, err := getFileDuration(path)
	if err != nil {
		log.Error("failed getting duration from mp3", "error", err)
	}
	return int(duration), nil
}

func getFileDuration(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return 0, err
	}
	defer streamer.Close()

	samples := streamer.Len()
	seconds := float64(samples) / float64(format.SampleRate)

	return seconds, nil
}
