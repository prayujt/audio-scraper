// Package youtube defines the audio-source port: locate a track on YouTube and
// download it. The concrete implementation lives in the impl subpackage.
package youtube

import "context"

// Provider locates and downloads audio for a given track.
type Provider interface {
	// Search returns a playable video URL for the given track. duration is the
	// expected track length in seconds (0 if unknown) and is used to rank
	// candidates; pass it to disambiguate between versions.
	Search(ctx context.Context, track, artist string, duration int) (string, error)
	// Download fetches the audio at videoURL to path and returns its duration
	// in seconds (-1 if the duration could not be determined).
	Download(ctx context.Context, path, videoURL string) (int, error)
}
