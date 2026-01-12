//go:build windows
// +build windows

package encode

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func EncodeFlac(inputPath string, outputPath string) error {
	// For Windows, we'll use a pure Go approach or download flac.exe
	// First, try to find flac.exe in PATH
	if _, err := exec.LookPath("flac"); err == nil {
		// flac.exe is available, use it
		cmd := exec.Command("flac", "-f", "-8", "-o", outputPath, inputPath)
		return cmd.Run()
	}

	// If flac.exe is not available, use pure Go encoding
	return encodeFlacPureGo(inputPath, outputPath)
}

// encodeFlacPureGo provides a fallback pure Go FLAC encoding
// This is a simplified version - you may want to use a library like github.com/go-audio/audio
func encodeFlacPureGo(inputPath string, outputPath string) error {
	// For now, return an error instructing the user to install FLAC
	// In a production version, you would use a Go FLAC library or embed flac.exe
	return fmt.Errorf("flac.exe not found in PATH. Please install FLAC from https://xiph.org/flac/download.html or run: winget install FLAC")
}

// readWAVData reads PCM data from a WAV file
func readWAVData(path string) ([]int16, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer file.Close()

	// Read RIFF header
	var riff [4]byte
	if _, err := io.ReadFull(file, riff[:]); err != nil {
		return nil, 0, 0, err
	}
	if string(riff[:]) != "RIFF" {
		return nil, 0, 0, fmt.Errorf("not a WAV file")
	}

	// Skip file size
	file.Seek(4, io.SeekCurrent)

	// Read WAVE
	var wave [4]byte
	if _, err := io.ReadFull(file, wave[:]); err != nil {
		return nil, 0, 0, err
	}
	if string(wave[:]) != "WAVE" {
		return nil, 0, 0, fmt.Errorf("not a WAV file")
	}

	// Find fmt and data chunks
	var channels, sampleRate int
	var dataSize int

	for {
		var chunkID [4]byte
		var chunkSize uint32

		if _, err := io.ReadFull(file, chunkID[:]); err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, 0, err
		}
		if err := binary.Read(file, binary.LittleEndian, &chunkSize); err != nil {
			return nil, 0, 0, err
		}

		switch string(chunkID[:]) {
		case "fmt ":
			var format, numChannels, sampleRateVal uint16
			binary.Read(file, binary.LittleEndian, &format)
			binary.Read(file, binary.LittleEndian, &numChannels)
			binary.Read(file, binary.LittleEndian, &sampleRateVal)
			channels = int(numChannels)
			sampleRate = int(sampleRateVal)
			file.Seek(int64(chunkSize-6), io.SeekCurrent)

		case "data":
			dataSize = int(chunkSize)
			// Read PCM data
			samples := make([]int16, dataSize/2)
			if err := binary.Read(file, binary.LittleEndian, &samples); err != nil {
				return nil, 0, 0, err
			}
			return samples, channels, sampleRate, nil

		default:
			file.Seek(int64(chunkSize), io.SeekCurrent)
		}
	}

	return nil, 0, 0, fmt.Errorf("no data chunk found")
}
