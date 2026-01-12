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
# Install letitrip (force clean install)
go clean -cache -modcache -i github.com/cloudboy-jh/letitrip/letitrip
go install github.com/cloudboy-jh/letitrip/letitrip@latest

# Install FLAC tools (required for encoding and tagging)
winget install Xiph.FLAC
```

If `letitrip` is not in your PATH, add `%USERPROFILE%\go\bin` to your system PATH or run:
```powershell
$env:Path += ";$env:USERPROFILE\go\bin"
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

## Troubleshooting

**"executable file not found" or "cdparanoia not found":**

This means `go install` grabbed an old cached version. Force a clean reinstall:

```powershell
# Remove cached version
go clean -i github.com/cloudboy-jh/letitrip/letitrip

# Clear Go module cache
go clean -modcache

# Reinstall from latest source
go install github.com/cloudboy-jh/letitrip/letitrip@latest

# Verify Windows version is installed
where letitrip
# Should show: C:\Users\<YourName>\go\bin\letitrip.exe
```

**"letitrip is not recognized":**

Add Go's bin directory to your PATH:
```powershell
$env:Path += ";$env:USERPROFILE\go\bin"
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
