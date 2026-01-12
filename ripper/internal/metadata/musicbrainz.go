package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/USERNAME/letitrip/ripper/internal/disc"
)

type discResponse struct {
	Releases []release `json:"releases"`
}

type release struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Date         string         `json:"date"`
	ArtistCredit []artistCredit `json:"artist-credit"`
	Media        []media        `json:"media"`
}

type artistCredit struct {
	Name string `json:"name"`
}

type media struct {
	Tracks []track `json:"tracks"`
}

type track struct {
	Number   string `json:"number"`
	Position int    `json:"position"`
	Title    string `json:"title"`
}

func FetchDiscMetadata(discID string) (Release, error) {
	url := fmt.Sprintf("https://musicbrainz.org/ws/2/discid/%s?inc=recordings+artists+artist-credits&fmt=json", discID)
	client := &http.Client{Timeout: 15 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("User-Agent", "letitrip/0.1 (https://github.com/USERNAME/letitrip)")

	response, err := client.Do(request)
	if err != nil {
		return Release{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("musicbrainz returned %s", response.Status)
	}

	var payload discResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Release{}, err
	}

	if len(payload.Releases) == 0 {
		return Release{}, fmt.Errorf("no release found")
	}

	selected := payload.Releases[0]
	release := Release{
		Artist:      artistName(selected.ArtistCredit),
		Album:       selected.Title,
		Year:        releaseYear(selected.Date),
		AlbumArtist: artistName(selected.ArtistCredit),
		ReleaseID:   selected.ID,
	}

	if len(selected.Media) > 0 {
		for _, tr := range selected.Media[0].Tracks {
			release.Tracks = append(release.Tracks, Track{Number: trackNumber(tr), Title: tr.Title})
		}
	}

	if release.ReleaseID != "" {
		if art, err := FetchCoverArt(release.ReleaseID); err == nil {
			release.CoverArt = art
		}
	}

	if len(release.Tracks) == 0 {
		return Release{}, fmt.Errorf("release has no tracks")
	}

	return release, nil
}

func FallbackMetadata(toc disc.TOC) Release {
	tracks := make([]Track, 0, len(toc.Tracks))
	for _, track := range toc.Tracks {
		tracks = append(tracks, Track{Number: track.Number, Title: fmt.Sprintf("Track %02d", track.Number)})
	}

	return Release{
		Artist:      "Unknown Artist",
		Album:       "Unknown Album",
		AlbumArtist: "Unknown Artist",
		Tracks:      tracks,
	}
}

func artistName(credits []artistCredit) string {
	if len(credits) == 0 {
		return "Unknown Artist"
	}
	return credits[0].Name
}

func releaseYear(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

func trackNumber(tr track) int {
	if tr.Position > 0 {
		return tr.Position
	}
	value, err := strconv.Atoi(strings.TrimSpace(tr.Number))
	if err != nil {
		return 0
	}
	return value
}
