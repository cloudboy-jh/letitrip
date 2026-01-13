package organize

type ProgressEventType string

const (
	EventInfo       ProgressEventType = "info"
	EventTrackStart ProgressEventType = "track_start"
	EventTrackStage ProgressEventType = "track_stage"
	EventTrackDone  ProgressEventType = "track_done"
	EventTrackFail  ProgressEventType = "track_fail"
	EventRetry      ProgressEventType = "retry"
	EventWarning    ProgressEventType = "warning"
	EventSummary    ProgressEventType = "summary"
)

type ProgressEvent struct {
	Type        ProgressEventType
	TrackNumber int
	Title       string
	Index       int
	Total       int
	Stage       string
	Speed       string
	Retries     int
	Message     string
	Error       string
	LogPath     string
}

type ProgressReporter interface {
	Report(event ProgressEvent)
}
