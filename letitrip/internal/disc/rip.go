package disc

import (
	"fmt"
	"os/exec"
)

func RipTrack(drive string, trackNumber int, outputPath string) error {
	cmd := exec.Command("cdparanoia", "-d", drive, fmt.Sprintf("%d", trackNumber), outputPath)
	return cmd.Run()
}
