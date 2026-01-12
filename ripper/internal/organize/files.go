package organize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudboy-jh/letitrip/ripper/internal/config"
	"github.com/cloudboy-jh/letitrip/ripper/internal/disc"
	"github.com/cloudboy-jh/letitrip/ripper/internal/encode"
	"github.com/cloudboy-jh/letitrip/ripper/internal/metadata"
)

func RipAndOrganize(drive string, cfg config.Config, toc disc.TOC, release metadata.Release) error {
	if len(release.Tracks) == 0 {
		return fmt.Errorf("no tracks available")
	}

	tempDir, err := os.MkdirTemp("", "letitrip")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	artistDir := sanitize(release.Artist)
	albumDir := sanitize(release.Album)
	outputDir := filepath.Join(cfg.OutputDir, artistDir, albumDir)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	trackMap := make(map[int]metadata.Track)
	for _, track := range release.Tracks {
		trackMap[track.Number] = track
	}

	for _, track := range toc.Tracks {
		trackMeta, ok := trackMap[track.Number]
		if !ok {
			trackMeta = metadata.Track{Number: track.Number, Title: fmt.Sprintf("Track %02d", track.Number)}
		}

		wavPath := filepath.Join(tempDir, fmt.Sprintf("track-%02d.wav", track.Number))
		if err := disc.RipTrack(drive, track.Number, wavPath); err != nil {
			return err
		}

		fileName := fmt.Sprintf("%02d - %s.flac", trackMeta.Number, sanitize(trackMeta.Title))
		flacPath := filepath.Join(outputDir, fileName)
		if err := encode.EncodeFlac(wavPath, flacPath); err != nil {
			return err
		}

		if err := metadata.ApplyTags(flacPath, release, trackMeta); err != nil {
			return err
		}
	}

	return nil
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	value = replacer.Replace(value)
	return strings.TrimSpace(value)
}
