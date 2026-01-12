package metadata

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func FetchCoverArt(releaseID string) ([]byte, error) {
	url := fmt.Sprintf("https://coverartarchive.org/release/%s/front", releaseID)
	client := &http.Client{Timeout: 15 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "letitrip/0.1 (https://github.com/USERNAME/letitrip)")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover art returned %s", response.Status)
	}

	return io.ReadAll(response.Body)
}
