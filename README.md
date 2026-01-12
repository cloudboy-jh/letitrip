![LetItRip logo](project-letitrip-logo.png)

# letitrip

Let it rip—turn your CDs into a personal Spotify powered by a Raspberry Pi.

## Overview

letitrip is a two-part system for ripping CD collections to lossless audio and streaming them on a home network.

- **Ripper (Windows CLI)**: Detects discs, fetches MusicBrainz metadata + cover art, and rips to FLAC.
- **Server (Raspberry Pi)**: Runs Navidrome in Docker for streaming to web, mobile, and Bluesound.

## Quick Start

### Windows Ripper

```powershell
go install github.com/cloudboy-jh/letitrip/letitrip@latest
letitrip
```

Requirements:

- Go 1.21+ (for installation)
- `cdparanoia` (CD ripping)
- `flac` + `metaflac` (encoding and tags)

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
