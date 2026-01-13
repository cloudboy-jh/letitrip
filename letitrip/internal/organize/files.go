package organize

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudboy-jh/letitrip/letitrip/internal/config"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/disc"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/encode"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/metadata"
)

const cddaBytesPerSecond = 176400

func RipAndOrganize(drive string, cfg config.Config, toc disc.TOC, release metadata.Release, reporter ProgressReporter) error {
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

	logger, logPath, closeLogger, err := newRipLogger(outputDir)
	if err != nil {
		return err
	}
	defer closeLogger()

	retryReporter := newRetryReporter(reporter, logger)

	trackMap := make(map[int]metadata.Track)
	for _, track := range release.Tracks {
		trackMap[track.Number] = track
	}

	var failedTracks []int
	successCount := 0
	totalTracks := len(toc.Tracks)

	for index, track := range toc.Tracks {
		trackMeta, ok := trackMap[track.Number]
		if !ok {
			trackMeta = metadata.Track{Number: track.Number, Title: fmt.Sprintf("Track %02d", track.Number)}
		}

		report(reporter, logger, ProgressEvent{
			Type:        EventTrackStart,
			TrackNumber: trackMeta.Number,
			Title:       trackMeta.Title,
			Index:       index + 1,
			Total:       totalTracks,
		})

		report(reporter, logger, ProgressEvent{
			Type:        EventTrackStage,
			TrackNumber: trackMeta.Number,
			Title:       trackMeta.Title,
			Stage:       "ripping",
		})

		wavPath := filepath.Join(tempDir, fmt.Sprintf("track-%02d.wav", track.Number))
		ripStats, err := disc.RipTrack(drive, track.Number, wavPath, retryReporter)
		if err != nil {
			report(reporter, logger, ProgressEvent{
				Type:        EventTrackFail,
				TrackNumber: trackMeta.Number,
				Title:       trackMeta.Title,
				Stage:       "ripping",
				Error:       err.Error(),
			})
			failedTracks = append(failedTracks, track.Number)
			continue
		}

		speed := formatSpeed(ripStats.Bytes, ripStats.Duration)

		report(reporter, logger, ProgressEvent{
			Type:        EventTrackStage,
			TrackNumber: trackMeta.Number,
			Title:       trackMeta.Title,
			Stage:       "encoding",
		})

		fileName := fmt.Sprintf("%02d - %s.flac", trackMeta.Number, sanitize(trackMeta.Title))
		flacPath := filepath.Join(outputDir, fileName)

		if err := encode.EncodeFlac(wavPath, flacPath); err != nil {
			report(reporter, logger, ProgressEvent{
				Type:        EventTrackFail,
				TrackNumber: trackMeta.Number,
				Title:       trackMeta.Title,
				Stage:       "encoding",
				Error:       err.Error(),
			})
			failedTracks = append(failedTracks, track.Number)
			continue
		}

		report(reporter, logger, ProgressEvent{
			Type:        EventTrackStage,
			TrackNumber: trackMeta.Number,
			Title:       trackMeta.Title,
			Stage:       "tagging",
		})

		if err := metadata.ApplyTags(flacPath, release, trackMeta); err != nil {
			report(reporter, logger, ProgressEvent{
				Type:        EventWarning,
				TrackNumber: trackMeta.Number,
				Title:       trackMeta.Title,
				Stage:       "tagging",
				Error:       err.Error(),
				Message:     "tagging failed; file left untagged",
			})
		}

		report(reporter, logger, ProgressEvent{
			Type:        EventTrackDone,
			TrackNumber: trackMeta.Number,
			Title:       trackMeta.Title,
			Stage:       "done",
			Speed:       speed,
			Retries:     ripStats.Retries,
			Message:     fileName,
		})
		successCount++
	}

	report(reporter, logger, ProgressEvent{
		Type:    EventSummary,
		Total:   totalTracks,
		Message: fmt.Sprintf("Ripping complete: %d/%d tracks successful", successCount, totalTracks),
		Error:   fmt.Sprintf("failed tracks: %v", failedTracks),
		LogPath: logPath,
	})

	if len(failedTracks) > 0 {
		return fmt.Errorf("failed to rip %d track(s): %v", len(failedTracks), failedTracks)
	}

	return nil
}

func formatSpeed(bytes int64, duration time.Duration) string {
	if bytes <= 0 || duration <= 0 {
		return ""
	}
	bytesPerSecond := float64(bytes) / duration.Seconds()
	speedX := bytesPerSecond / cddaBytesPerSecond
	mbPerSecond := bytesPerSecond / 1_000_000
	return fmt.Sprintf("%.2fx (%.2f MB/s)", speedX, mbPerSecond)
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

func report(reporter ProgressReporter, logger *log.Logger, event ProgressEvent) {
	if reporter != nil {
		reporter.Report(event)
	}
	if logger != nil {
		logger.Printf("event=%s track=%d title=%q stage=%s retries=%d speed=%s message=%q error=%q", event.Type, event.TrackNumber, event.Title, event.Stage, event.Retries, event.Speed, event.Message, event.Error)
	}
}
