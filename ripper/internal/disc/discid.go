package disc

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
)

func CalculateDiscID(toc TOC) (string, error) {
	if len(toc.Tracks) == 0 {
		return "", fmt.Errorf("no tracks available")
	}

	offsets := make([]int, 100)
	offsets[0] = toc.LeadoutLBA + 150
	for i, track := range toc.Tracks {
		offsets[i+1] = track.StartLBA + 150
	}

	firstTrack := toc.Tracks[0].Number
	lastTrack := toc.Tracks[len(toc.Tracks)-1].Number

	builder := make([]byte, 0, 1024)
	builder = append(builder, []byte(fmt.Sprintf("%02X", firstTrack))...)
	builder = append(builder, []byte(fmt.Sprintf("%02X", lastTrack))...)
	for i := 0; i < 100; i++ {
		builder = append(builder, []byte(fmt.Sprintf("%08X", offsets[i]))...)
	}

	hash := sha1.Sum(builder)
	encoding := base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._").WithPadding('-')
	return encoding.EncodeToString(hash[:]), nil
}
