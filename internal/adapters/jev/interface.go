// Package jev defines an optional, bounded candidate-ranking port.
package jev

import "context"

type Candidate struct {
	Key      string
	Title    string
	Uploader string
	Duration int
}

// Ranker selects one already safety-filtered candidate. It must never cause a
// download; callers retain deterministic validation and a heuristic fallback.
type Ranker interface {
	Rank(ctx context.Context, track, artist string, duration int, candidates []Candidate) (key string, confidence float64, err error)
}
