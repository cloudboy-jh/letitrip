//go:build !windows

package encode

import "os/exec"

func EncodeFlac(inputPath string, outputPath string) error {
	cmd := exec.Command("flac", "-f", "-o", outputPath, inputPath)
	return cmd.Run()
}
