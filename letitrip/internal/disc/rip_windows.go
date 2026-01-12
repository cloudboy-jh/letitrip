//go:build windows
// +build windows

package disc

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	IOCTL_CDROM_RAW_READ = 0x0002403E
	CD_SECTOR_SIZE       = 2352 // Raw CD audio sector size
	MAX_RETRIES          = 2    // Number of retry attempts for read errors
	RETRY_DELAY          = 500  // Milliseconds to wait between retries
)

type RAW_READ_INFO struct {
	DiskOffset  int64
	SectorCount uint32
	TrackMode   uint32
}

func RipTrack(drive string, trackNumber int, outputPath string) error {
	// First, read TOC to get track information
	toc, err := ReadTOC(drive)
	if err != nil {
		return fmt.Errorf("failed to read TOC: %v", err)
	}

	// Find the track
	var track *TrackTOC
	for i := range toc.Tracks {
		if toc.Tracks[i].Number == trackNumber {
			track = &toc.Tracks[i]
			break
		}
	}
	if track == nil {
		return fmt.Errorf("track %d not found", trackNumber)
	}

	// Open the CD drive
	devicePath := `\\.\` + drive[:1] + ":"
	handle, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr(devicePath),
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ,
		nil,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to open drive: %v", err)
	}
	defer syscall.CloseHandle(handle)

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Write WAV header
	if err := writeWAVHeader(outFile, track.LengthLBA); err != nil {
		return fmt.Errorf("failed to write WAV header: %v", err)
	}

	// Read and write audio data
	startLBA := track.StartLBA + 150 // Convert to absolute LBA
	sectorsToRead := track.LengthLBA

	const sectorsPerRead = 26 // Read ~60KB at a time
	buffer := make([]byte, sectorsPerRead*CD_SECTOR_SIZE)

	for sectorsRead := 0; sectorsRead < sectorsToRead; {
		currentSectors := sectorsPerRead
		if sectorsRead+currentSectors > sectorsToRead {
			currentSectors = sectorsToRead - sectorsRead
		}

		// Read sectors with retry logic
		bytesRead, err := readCDSectorsWithRetry(handle, startLBA+sectorsRead, currentSectors, buffer, trackNumber, sectorsRead)
		if err != nil {
			return fmt.Errorf("failed to read track %d after %d retries at sector %d: %v", trackNumber, MAX_RETRIES, sectorsRead, err)
		}

		// Write to output file
		if _, err := outFile.Write(buffer[:bytesRead]); err != nil {
			return fmt.Errorf("failed to write audio data: %v", err)
		}

		sectorsRead += currentSectors
	}

	return nil
}

// readCDSectorsWithRetry attempts to read CD sectors with retry logic for scratched discs
func readCDSectorsWithRetry(handle syscall.Handle, startLBA int, sectorCount int, buffer []byte, trackNumber int, sectorOffset int) (int, error) {
	var lastErr error

	for attempt := 0; attempt <= MAX_RETRIES; attempt++ {
		if attempt > 0 {
			// Log retry attempt to console
			fmt.Fprintf(os.Stderr, "⚠️  Track %d: Read error at sector %d (attempt %d/%d) - retrying...\n",
				trackNumber, sectorOffset, attempt, MAX_RETRIES)
			time.Sleep(RETRY_DELAY * time.Millisecond)
		}

		bytesRead, err := readCDSectors(handle, startLBA, sectorCount, buffer)
		if err == nil {
			if attempt > 0 {
				// Log successful retry
				fmt.Fprintf(os.Stderr, "✓ Track %d: Successfully read sector %d on attempt %d\n",
					trackNumber, sectorOffset, attempt+1)
			}
			return bytesRead, nil
		}

		lastErr = err
	}

	// All retries failed
	fmt.Fprintf(os.Stderr, "✗ Track %d: Failed to read sector %d after %d attempts: %v\n",
		trackNumber, sectorOffset, MAX_RETRIES+1, lastErr)
	return 0, lastErr
}

func readCDSectors(handle syscall.Handle, startLBA int, sectorCount int, buffer []byte) (int, error) {
	readInfo := RAW_READ_INFO{
		DiskOffset:  int64(startLBA) * 2048,
		SectorCount: uint32(sectorCount),
		TrackMode:   2, // CDDA (audio)
	}

	var bytesReturned uint32
	err := syscall.DeviceIoControl(
		handle,
		IOCTL_CDROM_RAW_READ,
		(*byte)(unsafe.Pointer(&readInfo)),
		uint32(unsafe.Sizeof(readInfo)),
		&buffer[0],
		uint32(len(buffer)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return 0, err
	}

	return int(bytesReturned), nil
}

func writeWAVHeader(w io.Writer, numSectors int) error {
	dataSize := uint32(numSectors * CD_SECTOR_SIZE)
	fileSize := dataSize + 36

	// RIFF header
	if err := binary.Write(w, binary.LittleEndian, []byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, fileSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, []byte("WAVE")); err != nil {
		return err
	}

	// fmt chunk
	if err := binary.Write(w, binary.LittleEndian, []byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil { // chunk size
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // audio format (PCM)
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(2)); err != nil { // channels
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(44100)); err != nil { // sample rate
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(176400)); err != nil { // byte rate (44100 * 2 * 2)
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(4)); err != nil { // block align (2 * 2)
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(16)); err != nil { // bits per sample
		return err
	}

	// data chunk
	if err := binary.Write(w, binary.LittleEndian, []byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	return nil
}
