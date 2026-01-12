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

func RipTrack(drive string, trackNumber int, outputPath string) error {
	var lastErr error

	for attempt := 0; attempt <= MAX_RETRIES; attempt++ {
		if attempt > 0 {
			// Log retry attempt to console
			fmt.Fprintf(os.Stderr, "⚠️  Track %d: Read error (attempt %d/%d) - retrying...\n",
				trackNumber, attempt, MAX_RETRIES)
			time.Sleep(RETRY_DELAY * time.Millisecond)
		}

		// cdparanoia has built-in error correction, but we still retry the entire track if it fails
		cmd := exec.Command("cdparanoia", "-d", drive, fmt.Sprintf("%d", trackNumber), outputPath)

		// Capture stderr to see cdparanoia's error messages
		output, err := cmd.CombinedOutput()

		if err == nil {
			if attempt > 0 {
				// Log successful retry
				fmt.Fprintf(os.Stderr, "✓ Track %d: Successfully read on attempt %d\n",
					trackNumber, attempt+1)
			}
			return nil
		}

		lastErr = fmt.Errorf("%v: %s", err, output)

		// Remove partial output file if it exists
		os.Remove(outputPath)
	}

	// All retries failed
	fmt.Fprintf(os.Stderr, "✗ Track %d: Failed after %d attempts: %v\n",
		trackNumber, MAX_RETRIES+1, lastErr)
	return lastErr
}
