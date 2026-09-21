package youtubeimpl

import (
	"context"
	"testing"

	"audio-scraper/internal/adapters/jev"
	"audio-scraper/internal/adapters/youtube"
	"audio-scraper/internal/logger"
)

func TestNewCandidateLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "configured value", input: 25, want: 25},
		{name: "zero uses default", input: 0, want: defaultCandidateLimit},
		{name: "negative uses default", input: -1, want: defaultCandidateLimit},
		{name: "maximum value", input: maxCandidateLimit, want: maxCandidateLimit},
		{name: "over maximum is clamped", input: maxCandidateLimit + 1, want: maxCandidateLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.input).candidateLimit; got != tt.want {
				t.Fatalf("candidateLimit = %d, want %d", got, tt.want)
			}
		})
	}
}

type rankingStub struct {
	key        string
	confidence float64
	called     bool
}

func (s *rankingStub) Rank(_ context.Context, _ string, _ string, _ int, candidates []jev.Candidate) (string, float64, error) {
	s.called = true
	if len(candidates) < 2 {
		return "", 0, nil
	}
	return s.key, s.confidence, nil
}

func TestRankWithJevPromotesSafeSelectedCandidate(t *testing.T) {
	stub := &rankingStub{key: "candidate_1", confidence: 0.91}
	client := New(25, stub)
	candidates := []youtube.Candidate{
		{URL: "https://www.youtube.com/watch?v=one", Title: "Artist - Track (Official Audio)", Uploader: "Artist - Topic", Duration: 180},
		{URL: "https://www.youtube.com/watch?v=two", Title: "Track", Uploader: "Label", Duration: 181},
		{URL: "https://www.youtube.com/watch?v=bad", Title: "Track (Live)", Uploader: "Artist", Duration: 180},
	}
	got := client.rankWithJev(context.Background(), logger.NewLogger(), "Track", "Artist", 180, candidates)
	if !stub.called {
		t.Fatal("expected Jev ranker to be called")
	}
	if got[0].URL != candidates[1].URL {
		t.Fatalf("selected candidate = %q, want %q", got[0].URL, candidates[1].URL)
	}
	if got[1].URL != candidates[0].URL || got[2].URL != candidates[2].URL {
		t.Fatalf("non-selected candidates did not retain order: %#v", got)
	}
}

func TestRankWithJevSkipsUnsafeOrDurationMismatchedCandidates(t *testing.T) {
	stub := &rankingStub{key: "candidate_0", confidence: 0.95}
	client := New(25, stub)
	candidates := []youtube.Candidate{
		{URL: "https://www.youtube.com/watch?v=live", Title: "Track (Live)", Uploader: "Artist", Duration: 180},
		{URL: "https://www.youtube.com/watch?v=loop", Title: "Track 1 Hour Loop", Uploader: "Artist", Duration: 3600},
		{URL: "https://www.youtube.com/watch?v=good", Title: "Track", Uploader: "Artist - Topic", Duration: 180},
	}
	got := client.rankWithJev(context.Background(), logger.NewLogger(), "Track", "Artist", 180, candidates)
	if stub.called {
		t.Fatal("Jev should not be called when fewer than two candidates pass deterministic filters")
	}
	if got[0].URL != candidates[0].URL {
		t.Fatal("heuristic order should be preserved on fallback")
	}
}
