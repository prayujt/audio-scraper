package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"

	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

type fsClient struct {
	musicHome string
}

func NewFSProvider(musicHome string) (ports.FSProvider, error) {
	if musicHome == "" {
		return nil, errors.New("missing MUSIC_HOME")
	}
	return &fsClient{
		musicHome: musicHome,
	}, nil
}

func (f *fsClient) InitializePath(ctx context.Context, job *models.DownloadJob) (string, error) {
	log := logger.From(ctx)
	path := filepath.Join(
		f.musicHome,
		job.Artist,
		job.Album,
	)

	if err := os.MkdirAll(path, 0755); err != nil {
		log.Error("failed to create directories", "path", path, "err", err)
		return "", errors.New("failed to create directories")
	}

	hasher := sha256.New()
	hasher.Write([]byte(job.Track))
	trackNameHash := hex.EncodeToString(hasher.Sum(nil))

	outputPath := filepath.Join(path, trackNameHash+".mp3")

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.Remove(outputPath); err != nil {
			return "", errors.New("failed to remove existing file")
		}
	}

	log.Info("initialized filesystem path", "output_path", outputPath)
	return outputPath, nil
}

func (f *fsClient) TagFile(ctx context.Context, filePath string, job *models.DownloadJob) error {
	log := logger.From(ctx)
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		log.Error("failed to open id3 tag", "err", err)
		return errors.New("open id3 tag failed")
	}
	defer tag.Close()
	tag.DeleteAllFrames()

	tag.SetTitle(job.Track)
	tag.SetArtist(job.Artist)
	tag.SetAlbum(job.Album)

	year := ""
	if job.ReleaseDate != "" {
		parts := strings.Split(job.ReleaseDate, "-")
		if len(parts) > 0 {
			year = parts[0]
		}
	}
	if year != "" {
		tag.SetYear(year)
	}

	if job.TrackNumber > 0 {
		tag.AddTextFrame("TRCK", tag.DefaultEncoding(), strconv.Itoa(job.TrackNumber))
	}

	if job.ThumbnailURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.ThumbnailURL, nil)
		if err != nil {
			log.Error("failed to create thumbnail request", "err", err)
			return errors.New("create thumbnail request failed")
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error("failed to fetch thumbnail", "err", err)
			return errors.New("fetch thumbnail failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error("unexpected thumbnail response status", "status", resp.StatusCode)
			return errors.New("fetch thumbnail failed")
		}

		imgData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("failed to read thumbnail data", "err", err)
			return errors.New("read thumbnail data failed")
		}

		mime := "image/jpeg"
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			if strings.HasPrefix(ct, "image/") {
				mime = strings.Split(ct, ";")[0]
			}
		}

		pic := id3v2.PictureFrame{
			Encoding:    tag.DefaultEncoding(),
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     imgData,
		}

		tag.AddAttachedPicture(pic)
	}

	if err := tag.Save(); err != nil {
		log.Error("failed to save id3 tag", "err", err)
		return errors.New("save id3 tag failed")
	}

	return nil
}
