package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cloudboy-jh/letitrip/letitrip/internal/metadata"
	"github.com/cloudboy-jh/letitrip/letitrip/internal/organize"
)

type trackView struct {
	number int
	title  string
	status string
	speed  string
	error  string
}

type model struct {
	release   metadata.Release
	discID    string
	events    <-chan organize.ProgressEvent
	tracks    []trackView
	trackIdx  map[int]int
	message   string
	summary   string
	logPath   string
	completed bool
}

type progressMsg struct {
	event organize.ProgressEvent
	ok    bool
}

func NewModel(release metadata.Release, discID string, events <-chan organize.ProgressEvent) tea.Model {
	tracks := make([]trackView, 0, len(release.Tracks))
	trackIdx := make(map[int]int)
	for index, track := range release.Tracks {
		tracks = append(tracks, trackView{number: track.Number, title: track.Title, status: "pending"})
		trackIdx[track.Number] = index
	}
	return model{release: release, discID: discID, events: events, tracks: tracks, trackIdx: trackIdx}
}

func (m model) Init() tea.Cmd {
	return waitForEvent(m.events)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch value := msg.(type) {
	case progressMsg:
		if !value.ok {
			return m, tea.Quit
		}
		m = m.applyEvent(value.event)
		if value.event.Type == organize.EventSummary {
			return m, tea.Quit
		}
		return m, waitForEvent(m.events)
	case tea.KeyMsg:
		if value.String() == "q" || value.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	builder := strings.Builder{}
	builder.WriteString("LetItRip Dashboard\n")
	builder.WriteString(fmt.Sprintf("Disc ID: %s\n", m.discID))
	builder.WriteString(fmt.Sprintf("Artist: %s\n", m.release.Artist))
	builder.WriteString(fmt.Sprintf("Album: %s\n", m.release.Album))
	if m.release.Year != "" {
		builder.WriteString(fmt.Sprintf("Year: %s\n", m.release.Year))
	}
	builder.WriteString("\n")

	for _, track := range m.tracks {
		status := statusLabel(track.status)
		speed := ""
		if track.speed != "" {
			speed = " " + track.speed
		}
		line := fmt.Sprintf("%02d %-40s [%s]%s", track.number, trimTitle(track.title, 40), status, speed)
		if track.error != "" {
			line = fmt.Sprintf("%s\n    error: %s", line, track.error)
		}
		builder.WriteString(line + "\n")
	}

	if m.message != "" {
		builder.WriteString("\n" + m.message + "\n")
	}
	if m.summary != "" {
		builder.WriteString("\n" + m.summary + "\n")
	}
	if m.logPath != "" {
		builder.WriteString("Log: " + m.logPath + "\n")
	}
	builder.WriteString("\nPress q to quit.\n")

	return builder.String()
}

func (m model) applyEvent(event organize.ProgressEvent) model {
	if event.TrackNumber != 0 {
		if index, ok := m.trackIdx[event.TrackNumber]; ok {
			track := m.tracks[index]
			switch event.Type {
			case organize.EventTrackStart:
				track.status = "ripping"
			case organize.EventTrackStage:
				track.status = event.Stage
			case organize.EventTrackDone:
				track.status = "done"
				track.speed = event.Speed
			case organize.EventTrackFail:
				track.status = "failed"
				track.error = event.Error
			}
			m.tracks[index] = track
		}
	}

	switch event.Type {
	case organize.EventRetry:
		m.message = fmt.Sprintf("Track %d retry: %s (attempt %d)", event.TrackNumber, event.Message, event.Retries)
	case organize.EventWarning:
		m.message = fmt.Sprintf("Warning: track %d %s", event.TrackNumber, event.Error)
	case organize.EventSummary:
		m.summary = event.Message
		m.logPath = event.LogPath
		m.completed = true
	case organize.EventInfo:
		if event.Message != "" {
			m.message = event.Message
		}
	}

	return m
}

func waitForEvent(events <-chan organize.ProgressEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		return progressMsg{event: event, ok: ok}
	}
}

func statusLabel(status string) string {
	switch status {
	case "ripping":
		return "RIP"
	case "encoding":
		return "ENC"
	case "tagging":
		return "TAG"
	case "done":
		return "OK"
	case "failed":
		return "FAIL"
	default:
		return "PEND"
	}
}

func trimTitle(title string, width int) string {
	runes := []rune(title)
	if len(runes) <= width {
		return title
	}
	return string(runes[:width-1]) + "…"
}
