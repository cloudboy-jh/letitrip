![LetItRip logo](project-letitrip-logo.png)

# letitrip

Let it rip—turn your CDs into a personal Spotify powered by a Raspberry Pi.

## Overview

letitrip is a two-part system for ripping CD collections to lossless audio and streaming them on a home network.

- **Ripper (Windows CLI)**: Detects discs, fetches MusicBrainz metadata + cover art, and rips to FLAC.
- **Server (Raspberry Pi)**: Runs Navidrome in Docker for streaming to web, mobile, and Bluesound.

## Quick Start

### Windows Ripper

**Installation:**

```powershell
# Install letitrip
go install github.com/cloudboy-jh/letitrip/letitrip@latest

# Install FLAC tools (required for encoding and tagging)
winget install FLAC
```

**Requirements:**

- Go 1.21+ (for installation only)
- FLAC tools (flac.exe + metaflac.exe) - installed via winget or from [https://xiph.org/flac/download.html](https://xiph.org/flac/download.html)

**Note:** The Windows version uses native Windows CD-ROM APIs for disc reading and ripping - no external CD ripping tools needed!

### Raspberry Pi Server

```bash
curl -fsSL https://raw.githubusercontent.com/cloudboy-jh/letitrip/main/server/setup.sh | bash
sudo mount /dev/sda1 /mnt/music
cd /opt/letitrip && docker compose up -d
```

## Usage

```bash
letitrip
letitrip --output D:/Music
letitrip --info
letitrip --yes
letitrip config --output D:/Music
```

## Output Structure

```
Music/
└── Artist/
    └── Album/
        ├── cover.jpg
        ├── 01 - Track.flac
        └── ...
```

## Repository Layout

```
letitrip/             # Go CLI
server/               # Navidrome setup
SPEC.md               # Full spec
```

## License

MIT. See `LICENSE`.
