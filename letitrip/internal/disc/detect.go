package disc

import (
	"errors"
	"os/exec"
	"strings"
)

func DetectDrive() (string, error) {
	cmd := exec.Command("wmic", "logicaldisk", "where", "DriveType=5", "get", "DeviceID")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ":") && len(line) == 2 {
			return line, nil
		}
	}

	return "", errors.New("no CD drive detected")
}
