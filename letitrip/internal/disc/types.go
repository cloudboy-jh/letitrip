package disc

import "time"

type TrackTOC struct {
	Number    int
	StartLBA  int
	LengthLBA int
}

type TOC struct {
	Tracks     []TrackTOC
	LeadoutLBA int
}

type RipStats struct {
	Bytes    int64
	Duration time.Duration
	Retries  int
}

type RetryScope string

type RetryEvent struct {
	TrackNumber int
	Attempt     int
	MaxAttempts int
	Sector      int
	Scope       RetryScope
	Err         error
}

type RetryReporter interface {
	OnRetry(event RetryEvent)
	OnRetrySuccess(event RetryEvent)
}
