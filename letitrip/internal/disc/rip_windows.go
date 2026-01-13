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
	MAX_TRACK_RETRIES    = 2    // Number of retry attempts for track reads
	MAX_SECTOR_RETRIES   = 2    // Number of retry attempts for sector reads
	RETRY_DELAY          = 500  // Milliseconds to wait between retries
)

type RAW_READ_INFO struct {
	DiskOffset  int64
	SectorCount uint32
	TrackMode   uint32
}

func RipTrack(drive string, trackNumber int, outputPath string, reporter RetryReporter) (RipStats, error) {
	start := time.Now()
	var lastErr error

	for attempt := 0; attempt <= MAX_TRACK_RETRIES; attempt++ {
		if attempt > 0 {
			if reporter != nil {
				reporter.OnRetry(RetryEvent{
					TrackNumber: trackNumber,
					Attempt:     attempt,
					MaxAttempts: MAX_TRACK_RETRIES,
					Scope:       "track",
				})
			}
			time.Sleep(time.Duration(attempt) * RETRY_DELAY * time.Millisecond)
		}

		bytesWritten, retries, err := ripTrackOnce(drive, trackNumber, outputPath, reporter)
		if err == nil {
			if attempt > 0 && reporter != nil {
				reporter.OnRetrySuccess(RetryEvent{
					TrackNumber: trackNumber,
					Attempt:     attempt,
					MaxAttempts: MAX_TRACK_RETRIES,
					Scope:       "track",
				})
			}
			return RipStats{Bytes: bytesWritten, Duration: time.Since(start), Retries: retries + attempt}, nil
		}

		lastErr = err
		_ = os.Remove(outputPath)
	}

	return RipStats{Duration: time.Since(start), Retries: MAX_TRACK_RETRIES}, fmt.Errorf("track %d failed after %d attempts: %w", trackNumber, MAX_TRACK_RETRIES+1, lastErr)
}

func ripTrackOnce(drive string, trackNumber int, outputPath string, reporter RetryReporter) (int64, int, error) {
	// First, read TOC to get track information
	toc, err := ReadTOC(drive)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read TOC: %w", err)
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
		return 0, 0, fmt.Errorf("track %d not found", trackNumber)
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
		return 0, 0, fmt.Errorf("failed to open drive %s: %w", devicePath, err)
	}
	defer syscall.CloseHandle(handle)

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Write WAV header
	if err := writeWAVHeader(outFile, track.LengthLBA); err != nil {
		return 0, 0, fmt.Errorf("failed to write WAV header: %w", err)
	}

	// Read and write audio data
	startLBA := track.StartLBA + 150 // Convert to absolute LBA
	sectorsToRead := track.LengthLBA

	const sectorsPerRead = 26 // Read ~60KB at a time
	buffer := make([]byte, sectorsPerRead*CD_SECTOR_SIZE)

	var bytesWritten int64
	var totalRetries int

	for sectorsRead := 0; sectorsRead < sectorsToRead; {
		currentSectors := sectorsPerRead
		if sectorsRead+currentSectors > sectorsToRead {
			currentSectors = sectorsToRead - sectorsRead
		}

		// Read sectors with retry logic
		bytesRead, retries, err := readCDSectorsWithRetry(handle, startLBA+sectorsRead, currentSectors, buffer, trackNumber, sectorsRead, reporter)
		totalRetries += retries
		if err != nil {
			return bytesWritten, totalRetries, fmt.Errorf("failed to read track %d at sector %d: %w", trackNumber, sectorsRead, err)
		}

		// Write to output file
		if _, err := outFile.Write(buffer[:bytesRead]); err != nil {
			return bytesWritten, totalRetries, fmt.Errorf("failed to write audio data: %w", err)
		}

		bytesWritten += int64(bytesRead)
		sectorsRead += currentSectors
	}

	return bytesWritten, totalRetries, nil
}

// readCDSectorsWithRetry attempts to read CD sectors with retry logic for scratched discs
func readCDSectorsWithRetry(handle syscall.Handle, startLBA int, sectorCount int, buffer []byte, trackNumber int, sectorOffset int, reporter RetryReporter) (int, int, error) {
	var lastErr error

	for attempt := 0; attempt <= MAX_SECTOR_RETRIES; attempt++ {
		if attempt > 0 {
			if reporter != nil {
				reporter.OnRetry(RetryEvent{
					TrackNumber: trackNumber,
					Attempt:     attempt,
					MaxAttempts: MAX_SECTOR_RETRIES,
					Sector:      sectorOffset,
					Scope:       "sector",
				})
			}
			time.Sleep(time.Duration(attempt) * RETRY_DELAY * time.Millisecond)
		}

		bytesRead, err := readCDSectors(handle, startLBA, sectorCount, buffer)
		if err == nil {
			if attempt > 0 && reporter != nil {
				reporter.OnRetrySuccess(RetryEvent{
					TrackNumber: trackNumber,
					Attempt:     attempt,
					MaxAttempts: MAX_SECTOR_RETRIES,
					Sector:      sectorOffset,
					Scope:       "sector",
				})
			}
			return bytesRead, attempt, nil
		}

		lastErr = fmt.Errorf("sector %d lba %d read failed: %w", sectorOffset, startLBA, err)
	}

	return 0, MAX_SECTOR_RETRIES, lastErr
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
