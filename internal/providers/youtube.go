package providers

import (
	"context"
	"errors"
	"os/exec"

	"audio-scraper/internal/logger"
	"audio-scraper/internal/ports"
)

type youtubeClient struct{}

func NewYTProvider() ports.YTProvider {
	return &youtubeClient{}
}

func (y *youtubeClient) Search(ctx context.Context, track string, album string, artist string) (string, error) {
	log := logger.From(ctx)

	log.Info("performing yt search", "track", track, "album", album, "artist", artist)
	cmd := exec.Command("python3", "scripts/yt-music.py", track, album, artist)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("yt search command failed", "err", err, "output", string(output))
		return "", errors.New("yt search failed")
	}

	log.Info("yt search output", "output", string(output))
	return string(output), nil
}

func (y *youtubeClient) Download(ctx context.Context, path string, videoURL string) error {
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
		log.Error("yt-dlp command failed", "err", err, "output", string(output))
		return errors.New("yt-dlp download failed")
	}

	return nil
}
