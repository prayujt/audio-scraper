// Package filesystemimpl implements the filesystem.Provider port: it writes
// downloaded audio into a MUSIC_HOME/Artist/Album tree and tags the files.
package filesystemimpl

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

	"audio-scraper/internal/adapters/filesystem"
	"audio-scraper/internal/adapters/lrclib"
	"audio-scraper/internal/logger"
	"audio-scraper/internal/models"
)

// Client is the local-filesystem library provider.
type Client struct {
	musicHome string
	lrc       lrclib.Provider
}

var _ filesystem.Provider = (*Client)(nil)

// New returns a filesystem provider rooted at musicHome, using lrc to resolve
// lyrics during tagging.
func New(musicHome string, lrc lrclib.Provider) (*Client, error) {
	if musicHome == "" {
		return nil, errors.New("missing MUSIC_HOME")
	}
	return &Client{
		musicHome: musicHome,
		lrc:       lrc,
	}, nil
}

// outputPath returns the deterministic on-disk path for a track:
// MUSIC_HOME/Artist/Album/sha256(Track).mp3.
func (f *Client) outputPath(artist, album, track string) string {
	hasher := sha256.New()
	hasher.Write([]byte(track))
	trackNameHash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(f.musicHome, artist, album, trackNameHash+".mp3")
}

func (f *Client) InitializePath(ctx context.Context, job *models.DownloadJob) (string, error) {
	log := logger.From(ctx)
	dir := filepath.Join(f.musicHome, job.Artist, job.Album)

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error("failed to create directories", "path", dir, "error", err)
		return "", errors.New("failed to create directories")
	}

	outputPath := f.outputPath(job.Artist, job.Album, job.Track)

	if _, err := os.Stat(outputPath); err == nil {
		if err := os.Remove(outputPath); err != nil {
			return "", errors.New("failed to remove existing file")
		}
	}

	log.Info("initialized filesystem path", "output_path", outputPath)
	return outputPath, nil
}

// ReplaceAudio re-downloads the audio for an existing file while keeping its
// tags. It downloads to a sibling temp file, copies the original's ID3 frames
// onto it, then atomically renames it over the original. The original is only
// touched once the new file is fully prepared.
func (f *Client) ReplaceAudio(ctx context.Context, job *models.DownloadJob, download func(ctx context.Context, dest string) error) error {
	log := logger.From(ctx)
	path := f.outputPath(job.Artist, job.Album, job.Track)
	log = log.With("path", path)

	if _, err := os.Stat(path); err != nil {
		log.Error("existing file not found for replacement", "error", err)
		return errors.New("existing file not found")
	}

	// Capture the existing tags before we touch anything.
	frames, err := readFrames(path)
	if err != nil {
		log.Error("failed to read existing tags", "error", err)
		return errors.New("failed to read existing tags")
	}

	tmp := filepath.Join(filepath.Dir(path), ".replace-"+filepath.Base(path))
	_ = os.Remove(tmp) // clear any stale temp from a prior failed run

	log.Info("downloading replacement audio")
	if err := download(ctx, tmp); err != nil {
		log.Error("replacement download failed", "error", err)
		os.Remove(tmp)
		return errors.New("replacement download failed")
	}

	if err := writeFrames(tmp, frames); err != nil {
		log.Error("failed to restore tags", "error", err)
		os.Remove(tmp)
		return errors.New("failed to restore tags")
	}

	if err := os.Rename(tmp, path); err != nil {
		log.Error("failed to replace original file", "error", err)
		os.Remove(tmp)
		return errors.New("failed to replace original file")
	}

	log.Info("replaced audio, preserved existing tags")
	return nil
}

// readFrames reads all ID3 frames from an mp3 file.
func readFrames(path string) (map[string][]id3v2.Framer, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return nil, err
	}
	defer tag.Close()

	frames := make(map[string][]id3v2.Framer)
	for id, fs := range tag.AllFrames() {
		frames[id] = append([]id3v2.Framer(nil), fs...)
	}
	return frames, nil
}

// writeFrames replaces all ID3 frames on an mp3 file with the given frames.
func writeFrames(path string, frames map[string][]id3v2.Framer) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.DeleteAllFrames()
	for id, fs := range frames {
		for _, fr := range fs {
			tag.AddFrame(id, fr)
		}
	}
	return tag.Save()
}

func (f *Client) TagFile(ctx context.Context, filePath string, job *models.DownloadJob) error {
	log := logger.From(ctx)
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		log.Error("failed to open id3 tag", "error", err)
		return errors.New("open id3 tag failed")
	}
	defer tag.Close()
	tag.DeleteAllFrames()

	tag.SetTitle(job.Track)
	tag.SetArtist(job.Artist)
	tag.SetAlbum(job.Album)

	lyrics, err := f.lrc.FindLyrics(ctx, &lrclib.LrclibRequest{
		Artist:   job.Artist,
		Album:    job.Album,
		Track:    job.Track,
		Duration: job.Duration,
	})
	if err != nil {
		log.Error("failed to find lyrics", "error", err)
	}

	if lyrics != nil && lyrics.Segments == nil {
		if lyrics.Full != nil {
			uslt := id3v2.UnsynchronisedLyricsFrame{
				Encoding:          id3v2.EncodingUTF8,
				Language:          "eng",
				ContentDescriptor: "",
				Lyrics:            *lyrics.Full,
			}
			tag.AddUnsynchronisedLyricsFrame(uslt)
		}
	} else if lyrics != nil && lyrics.Segments != nil {
		lyricsText := lyrics.SyncedToText()
		uslt := id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: "",
			Lyrics:            lyricsText,
		}
		tag.AddUnsynchronisedLyricsFrame(uslt)
	}

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
			log.Error("failed to create thumbnail request", "error", err)
			return errors.New("create thumbnail request failed")
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error("failed to fetch thumbnail", "error", err)
			return errors.New("fetch thumbnail failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error("unexpected thumbnail response status", "status", resp.StatusCode)
			return errors.New("fetch thumbnail failed")
		}

		imgData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("failed to read thumbnail data", "error", err)
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
		log.Error("failed to save id3 tag", "error", err)
		return errors.New("save id3 tag failed")
	}

	return nil
}
