package organize

import (
	"fmt"
	"io"
)

type CLIReporter struct {
	out io.Writer
	err io.Writer
}

func NewCLIReporter(out io.Writer, err io.Writer) ProgressReporter {
	return CLIReporter{out: out, err: err}
}

func (r CLIReporter) Report(event ProgressEvent) {
	switch event.Type {
	case EventTrackStart:
		fmt.Fprintf(r.out, "\nRipping track %d/%d: %s\n", event.Index, event.Total, event.Title)
	case EventTrackStage:
		if event.Stage == "encoding" {
			fmt.Fprintln(r.out, "  Encoding to FLAC...")
		}
		if event.Stage == "tagging" {
			fmt.Fprintln(r.out, "  Adding metadata tags...")
		}
	case EventTrackDone:
		suffix := ""
		if event.Speed != "" {
			suffix = fmt.Sprintf(" (%s)", event.Speed)
		}
		fmt.Fprintf(r.out, "Track %d complete: %s%s\n", event.TrackNumber, event.Message, suffix)
	case EventTrackFail:
		fmt.Fprintf(r.err, "Failed to rip track %d (%s) during %s: %s\n", event.TrackNumber, event.Title, event.Stage, event.Error)
	case EventWarning:
		fmt.Fprintf(r.err, "Warning: track %d (%s) %s: %s\n", event.TrackNumber, event.Title, event.Stage, event.Error)
	case EventRetry:
		fmt.Fprintf(r.err, "Track %d retry: %s (attempt %d)\n", event.TrackNumber, event.Message, event.Retries)
	case EventInfo:
		if event.Message != "" {
			fmt.Fprintln(r.out, event.Message)
		}
	case EventSummary:
		fmt.Fprintln(r.out, "\n=======================================")
		fmt.Fprintln(r.out, event.Message)
		if event.Error != "failed tracks: []" {
			fmt.Fprintln(r.err, event.Error)
		}
		if event.LogPath != "" {
			fmt.Fprintf(r.out, "Log: %s\n", event.LogPath)
		}
	}
}
