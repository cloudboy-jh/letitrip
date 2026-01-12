//go:build !windows

package disc

import (
	"bufio"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	trackLine = regexp.MustCompile(`track\s+(\d+):\s+(\d+)\s+\[[^\]]+\]\s+(\d+)\s+\[[^\]]+\]`)
	totalLine = regexp.MustCompile(`TOTAL\s+(\d+)\s+\[[^\]]+\]`)
)

func ReadTOC(drive string) (TOC, error) {
	cmd := exec.Command("cdparanoia", "-d", drive, "-Q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return TOC{}, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var tracks []TrackTOC
	var leadout int
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if match := trackLine.FindStringSubmatch(line); match != nil {
			trackNum, _ := strconv.Atoi(match[1])
			length, _ := strconv.Atoi(match[2])
			start, _ := strconv.Atoi(match[3])
			tracks = append(tracks, TrackTOC{Number: trackNum, StartLBA: start, LengthLBA: length})
			continue
		}
		if match := totalLine.FindStringSubmatch(line); match != nil {
			leadout, _ = strconv.Atoi(match[1])
		}
	}

	if len(tracks) == 0 {
		return TOC{}, errors.New("no tracks detected")
	}

	if leadout == 0 {
		last := tracks[len(tracks)-1]
		leadout = last.StartLBA + last.LengthLBA
	}

	return TOC{Tracks: tracks, LeadoutLBA: leadout}, nil
}
