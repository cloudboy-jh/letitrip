package organize

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudboy-jh/letitrip/letitrip/internal/disc"
)

type retryReporter struct {
	reporter ProgressReporter
	logger   *log.Logger
}

func newRetryReporter(reporter ProgressReporter, logger *log.Logger) disc.RetryReporter {
	if reporter == nil && logger == nil {
		return nil
	}
	return retryReporter{reporter: reporter, logger: logger}
}

func (r retryReporter) OnRetry(event disc.RetryEvent) {
	message := fmt.Sprintf("retrying %s read", event.Scope)
	if event.Scope == "sector" {
		message = fmt.Sprintf("retrying sector read at %d", event.Sector)
	}
	report(r.reporter, r.logger, ProgressEvent{
		Type:        EventRetry,
		TrackNumber: event.TrackNumber,
		Stage:       string(event.Scope),
		Retries:     event.Attempt,
		Message:     message,
		Error:       errorString(event.Err),
	})
}

func (r retryReporter) OnRetrySuccess(event disc.RetryEvent) {
	message := fmt.Sprintf("%s read recovered after retry", event.Scope)
	if event.Scope == "sector" {
		message = fmt.Sprintf("sector %d read recovered", event.Sector)
	}
	report(r.reporter, r.logger, ProgressEvent{
		Type:        EventInfo,
		TrackNumber: event.TrackNumber,
		Stage:       string(event.Scope),
		Retries:     event.Attempt,
		Message:     message,
	})
}

func newRipLogger(outputDir string) (*log.Logger, string, func(), error) {
	fileName := fmt.Sprintf("letitrip-%s.log", time.Now().Format("20060102-150405"))
	path := filepath.Join(outputDir, fileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, "", func() {}, err
	}
	logger := log.New(file, "", log.LstdFlags)
	return logger, path, func() { _ = file.Close() }, nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
