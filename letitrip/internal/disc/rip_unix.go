//go:build !windows

package disc

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	MAX_RETRIES = 2   // Number of retry attempts for read errors
	RETRY_DELAY = 500 // Milliseconds to wait between retries
)

func RipTrack(drive string, trackNumber int, outputPath string, reporter RetryReporter) (RipStats, error) {
	start := time.Now()
	var lastErr error

	for attempt := 0; attempt <= MAX_RETRIES; attempt++ {
		if attempt > 0 {
			if reporter != nil {
				reporter.OnRetry(RetryEvent{
					TrackNumber: trackNumber,
					Attempt:     attempt,
					MaxAttempts: MAX_RETRIES,
					Scope:       "track",
				})
			}
			time.Sleep(time.Duration(attempt) * RETRY_DELAY * time.Millisecond)
		}

		// cdparanoia has built-in error correction, but we still retry the entire track if it fails
		cmd := exec.Command("cdparanoia", "-d", drive, fmt.Sprintf("%d", trackNumber), outputPath)

		// Capture stderr to see cdparanoia's error messages
		output, err := cmd.CombinedOutput()

		if err == nil {
			if attempt > 0 && reporter != nil {
				reporter.OnRetrySuccess(RetryEvent{
					TrackNumber: trackNumber,
					Attempt:     attempt,
					MaxAttempts: MAX_RETRIES,
					Scope:       "track",
				})
			}
			var bytes int64
			if info, statErr := os.Stat(outputPath); statErr == nil {
				bytes = info.Size()
			}
			return RipStats{Bytes: bytes, Duration: time.Since(start), Retries: attempt}, nil
		}

		lastErr = fmt.Errorf("%v: %s", err, output)

		// Remove partial output file if it exists
		_ = os.Remove(outputPath)
	}

	return RipStats{Duration: time.Since(start), Retries: MAX_RETRIES}, fmt.Errorf("track %d failed after %d attempts: %w", trackNumber, MAX_RETRIES+1, lastErr)
}
