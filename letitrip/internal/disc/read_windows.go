//go:build windows
// +build windows

package disc

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	IOCTL_CDROM_READ_TOC = 0x00024000
	TRACK_MODE_AUDIO     = 0
)

type CDROM_TOC struct {
	Length     [2]byte
	FirstTrack byte
	LastTrack  byte
	TrackData  [100]TRACK_DATA
}

type TRACK_DATA struct {
	Reserved    byte
	Control     byte
	TrackNumber byte
	Reserved1   byte
	Address     [4]byte
}

func ReadTOC(drive string) (TOC, error) {
	if len(drive) < 2 || drive[1] != ':' {
		return TOC{}, fmt.Errorf("invalid drive format: %s", drive)
	}

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
		return TOC{}, fmt.Errorf("failed to open drive: %v", err)
	}
	defer syscall.CloseHandle(handle)

	var toc CDROM_TOC
	var bytesReturned uint32

	err = syscall.DeviceIoControl(
		handle,
		IOCTL_CDROM_READ_TOC,
		nil,
		0,
		(*byte)(unsafe.Pointer(&toc)),
		uint32(unsafe.Sizeof(toc)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return TOC{}, fmt.Errorf("failed to read TOC: %v", err)
	}

	if toc.FirstTrack == 0 || toc.LastTrack == 0 {
		return TOC{}, errors.New("no tracks detected")
	}

	var tracks []TrackTOC
	numTracks := int(toc.LastTrack - toc.FirstTrack + 1)

	for i := 0; i < numTracks; i++ {
		trackData := toc.TrackData[i]
		if trackData.TrackNumber == 0 {
			break
		}

		startLBA := msf2lba(
			int(trackData.Address[1]),
			int(trackData.Address[2]),
			int(trackData.Address[3]),
		)

		var lengthLBA int
		if i < numTracks-1 {
			nextTrackData := toc.TrackData[i+1]
			nextLBA := msf2lba(
				int(nextTrackData.Address[1]),
				int(nextTrackData.Address[2]),
				int(nextTrackData.Address[3]),
			)
			lengthLBA = nextLBA - startLBA
		} else {
			// For the last track, use the leadout position
			leadoutData := toc.TrackData[numTracks]
			leadoutLBA := msf2lba(
				int(leadoutData.Address[1]),
				int(leadoutData.Address[2]),
				int(leadoutData.Address[3]),
			)
			lengthLBA = leadoutLBA - startLBA
		}

		tracks = append(tracks, TrackTOC{
			Number:    int(trackData.TrackNumber),
			StartLBA:  startLBA,
			LengthLBA: lengthLBA,
		})
	}

	if len(tracks) == 0 {
		return TOC{}, errors.New("no tracks detected")
	}

	// Get leadout position (stored after last track)
	leadoutData := toc.TrackData[numTracks]
	leadoutLBA := msf2lba(
		int(leadoutData.Address[1]),
		int(leadoutData.Address[2]),
		int(leadoutData.Address[3]),
	)

	return TOC{
		Tracks:     tracks,
		LeadoutLBA: leadoutLBA,
	}, nil
}

// msf2lba converts Minutes:Seconds:Frames to Logical Block Address
func msf2lba(m, s, f int) int {
	return (m * 60 * 75) + (s * 75) + f - 150
}
