//go:build !windows

package metadata

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func ApplyTags(filePath string, release Release, track Track) error {
	args := []string{
		"--set-tag=ARTIST=" + release.Artist,
		"--set-tag=ALBUM=" + release.Album,
		"--set-tag=TITLE=" + track.Title,
		"--set-tag=TRACKNUMBER=" + fmt.Sprintf("%02d", track.Number),
		"--set-tag=TRACKTOTAL=" + fmt.Sprintf("%02d", len(release.Tracks)),
	}

	if release.Year != "" {
		args = append(args, "--set-tag=DATE="+release.Year)
	}
	if release.Genre != "" {
		args = append(args, "--set-tag=GENRE="+release.Genre)
	}
	if release.AlbumArtist != "" {
		args = append(args, "--set-tag=ALBUMARTIST="+release.AlbumArtist)
	}

	args = append(args, filePath)
	cmd := exec.Command("metaflac", args...)
	if err := cmd.Run(); err != nil {
		return err
	}

	if len(release.CoverArt) > 0 {
		coverPath := filepath.Join(filepath.Dir(filePath), "cover.jpg")
		if err := os.WriteFile(coverPath, release.CoverArt, 0o644); err != nil {
			return err
		}
		if err := exec.Command("metaflac", "--import-picture-from="+coverPath, filePath).Run(); err != nil {
			return err
		}
	}

	return nil
}
