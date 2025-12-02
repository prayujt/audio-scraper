package providers

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/faiface/beep/mp3"

	"audio-scraper/internal/logger"
)

type youtubeClient struct{}

type YTProvider interface {
	Search(ctx context.Context, track string, album string, artist string) (string, error)
	Download(ctx context.Context, path string, videoURL string) (int, error)
}

func NewYTProvider() YTProvider {
	return &youtubeClient{}
}

func (y *youtubeClient) Search(ctx context.Context, track string, album string, artist string) (string, error) {
	log := logger.From(ctx)

	log.Info("performing yt search", "track", track, "album", album, "artist", artist)
	cmd := exec.Command("python3", "scripts/yt-music.py", track, album, artist)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("yt search command failed", "error", err, "output", string(output))
		return "", errors.New("yt search failed")
	}

	log.Info("yt search output", "output", string(output))
	return string(output), nil
}

func (y *youtubeClient) Download(ctx context.Context, path string, videoURL string) (int, error) {
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
