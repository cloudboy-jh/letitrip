package organize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudboy-jh/letitrip/letitrip/internal/config"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/disc"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/encode"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/metadata"
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

	var failedTracks []int
	successCount := 0
	totalTracks := len(toc.Tracks)

	for _, track := range toc.Tracks {
		trackMeta, ok := trackMap[track.Number]
		if !ok {
			trackMeta = metadata.Track{Number: track.Number, Title: fmt.Sprintf("Track %02d", track.Number)}
		}

		fmt.Fprintf(os.Stdout, "\n🎵 Ripping track %d/%d: %s\n", track.Number, totalTracks, trackMeta.Title)

		wavPath := filepath.Join(tempDir, fmt.Sprintf("track-%02d.wav", track.Number))
		if err := disc.RipTrack(drive, track.Number, wavPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to rip track %d (%s): %v\n", track.Number, trackMeta.Title, err)
			failedTracks = append(failedTracks, track.Number)
			continue
		}

		fileName := fmt.Sprintf("%02d - %s.flac", trackMeta.Number, sanitize(trackMeta.Title))
		flacPath := filepath.Join(outputDir, fileName)

		fmt.Fprintf(os.Stdout, "   Encoding to FLAC...\n")
		if err := encode.EncodeFlac(wavPath, flacPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to encode track %d (%s): %v\n", track.Number, trackMeta.Title, err)
			failedTracks = append(failedTracks, track.Number)
			continue
		}

		fmt.Fprintf(os.Stdout, "   Adding metadata tags...\n")
		if err := metadata.ApplyTags(flacPath, release, trackMeta); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to tag track %d (%s): %v\n", track.Number, trackMeta.Title, err)
			// Don't mark as failed - the file is still usable without tags
		}

		fmt.Fprintf(os.Stdout, "✓ Track %d complete: %s\n", track.Number, fileName)
		successCount++
	}

	// Print summary
	fmt.Fprintf(os.Stdout, "\n═══════════════════════════════════════\n")
	fmt.Fprintf(os.Stdout, "Ripping complete: %d/%d tracks successful\n", successCount, totalTracks)

	if len(failedTracks) > 0 {
		fmt.Fprintf(os.Stderr, "\n⚠️  Failed tracks: %v\n", failedTracks)
		fmt.Fprintf(os.Stderr, "These tracks may be scratched or unreadable.\n")
		fmt.Fprintf(os.Stderr, "Try cleaning the disc and running again.\n")
		return fmt.Errorf("failed to rip %d track(s): %v", len(failedTracks), failedTracks)
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
