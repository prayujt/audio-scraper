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
	"sort"
	"strings"
	"time"

	"github.com/faiface/beep/mp3"

	"audio-scraper/internal/adapters/jev"
	"audio-scraper/internal/adapters/youtube"
	"audio-scraper/internal/logger"
)

const (
	defaultCandidateLimit = 25
	maxCandidateLimit     = 50
)

// Client is the YouTube-backed audio provider.
type Client struct {
	candidateLimit int
	ranker         jev.Ranker
}

var _ youtube.Provider = (*Client)(nil)

// New returns a YouTube audio provider. The configured candidate limit is
// clamped so a bad deployment value cannot turn a normal search into an
// unbounded yt-dlp request.
func New(candidateLimit int, rankers ...jev.Ranker) *Client {
	if candidateLimit < 1 {
		candidateLimit = defaultCandidateLimit
	}
	if candidateLimit > maxCandidateLimit {
		candidateLimit = maxCandidateLimit
	}
	var ranker jev.Ranker
	if len(rankers) > 0 {
		ranker = rankers[0]
	}
	return &Client{candidateLimit: candidateLimit, ranker: ranker}
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
	candidates, err := y.Candidates(ctx, track, artist, duration)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", errors.New("yt search returned no usable results")
	}
	best := candidates[0]
	logger.From(ctx).Info("yt search selected", "url", best.URL, "title", best.Title)
	return best.URL, nil
}

// Candidates runs a ytsearch and returns the entries ranked by score, best
// first. It mirrors the ranking used by Search but exposes the full list for
// manual selection.
func (y *Client) Candidates(ctx context.Context, track, artist string, duration int) ([]youtube.Candidate, error) {
	log := logger.From(ctx)
	query := strings.TrimSpace(track + " " + artist)
	log.Info("performing yt search", "query", query, "duration", duration)

	cmd := exec.CommandContext(
		ctx,
		"yt-dlp",
		"--no-warnings",
		"--flat-playlist",
		"-J",
		fmt.Sprintf("ytsearch%d:%s", y.candidateLimit, query),
	)
	out, err := cmd.Output()
	if err != nil {
		log.Error("yt search command failed", "error", err)
		return nil, errors.New("yt search failed")
	}

	var res ytSearchResponse
	if err := json.Unmarshal(out, &res); err != nil {
		log.Error("failed to parse yt search output", "error", err)
		return nil, errors.New("yt search failed")
	}

	type scored struct {
		entry ytEntry
		score float64
	}
	var ranked []scored
	for _, e := range res.Entries {
		if e.ID == "" {
			continue
		}
		ranked = append(ranked, scored{entry: e, score: score(track, artist, duration, e)})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	candidates := make([]youtube.Candidate, 0, len(ranked))
	for _, r := range ranked {
		uploader := r.entry.Uploader
		if uploader == "" {
			uploader = r.entry.Channel
		}
		candidates = append(candidates, youtube.Candidate{URL: "https://www.youtube.com/watch?v=" + r.entry.ID, Title: r.entry.Title, Uploader: uploader, Duration: int(r.entry.Duration)})
	}
	return y.rankWithJev(ctx, log, track, artist, duration, candidates), nil
}

// rankWithJev only reorders candidates that pass conservative deterministic
// checks. Any timeout, low-confidence answer, or unsafe set preserves the
// existing heuristic ordering and never blocks downloads.
func (y *Client) rankWithJev(ctx context.Context, log logger.Logger, track, artist string, duration int, candidates []youtube.Candidate) []youtube.Candidate {
	if y.ranker == nil || len(candidates) < 2 {
		return candidates
	}
	eligible := make([]jev.Candidate, 0, len(candidates))
	indices := make(map[string]int)
	for i, candidate := range candidates {
		if !safeForJev(track, artist, duration, candidate) {
			continue
		}
		key := fmt.Sprintf("candidate_%d", i)
		eligible = append(eligible, jev.Candidate{Key: key, Title: candidate.Title, Uploader: candidate.Uploader, Duration: candidate.Duration})
		indices[key] = i
	}
	if len(eligible) < 2 {
		return candidates
	}
	decisionCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	key, confidence, err := y.ranker.Rank(decisionCtx, track, artist, duration, eligible)
	if err != nil {
		log.Info("Jev candidate ranking fallback", "eligible", len(eligible), "error", err)
		return candidates
	}
	i := indices[key]
	selected := candidates[i]
	ordered := make([]youtube.Candidate, 0, len(candidates))
	ordered = append(ordered, selected)
	for j, candidate := range candidates {
		if j != i {
			ordered = append(ordered, candidate)
		}
	}
	log.Info("Jev candidate ranking selected", "candidate", selected.URL, "confidence", confidence, "eligible", len(eligible))
	return ordered
}

func safeForJev(track, artist string, expectedDuration int, candidate youtube.Candidate) bool {
	haystack := normalize(candidate.Title + " " + candidate.Uploader)
	for _, banned := range []string{" cover ", " karaoke ", " live ", " remix ", " slowed ", " reverb ", " sped up ", " loop ", " reaction ", " instrumental ", " piano ", " arrangement ", " tribute "} {
		if strings.Contains(" "+haystack+" ", banned) {
			return false
		}
	}
	if expectedDuration > 0 && candidate.Duration > 0 {
		tolerance := int(math.Max(5, math.Ceil(float64(expectedDuration)*0.03)))
		if int(math.Abs(float64(candidate.Duration-expectedDuration))) > tolerance {
			return false
		}
	}
	return normalize(track) != "" && normalize(artist) != ""
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
