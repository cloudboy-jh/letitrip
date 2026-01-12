# letitrip

> Let it rip—turn your CDs into a personal Spotify powered by a Raspberry Pi.

---

## Overview

letitrip is a two-part system for ripping CD collections to lossless audio and streaming them on a home network.

**Part 1: Ripper** — A Windows CLI tool that rips CDs to FLAC with full metadata and album art, organized alphabetically by artist.

**Part 2: Server** — A Raspberry Pi 5 (Pironman 5) running Navidrome, serving your music library to any device on your network, including Bluesound speakers.

---

## Goals

- One-command CD ripping with automatic metadata lookup
- Lossless audio quality (FLAC)
- Clean file organization: `Artist/Album/01 - Track.flac`
- Embedded metadata: artist, album, track name, track number, year, album art
- Lightweight streaming server that runs on a Pi
- Mobile/web access to the full library
- Bluesound compatibility via DLNA/UPnP

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         RIPPING STATION                         │
│                         (Windows PC)                            │
│                                                                 │
│   ┌─────────┐      ┌─────────────┐      ┌─────────────────┐    │
│   │ LG USB  │ ──▶  │  letitrip   │ ──▶  │ External HDD    │    │
│   │ CD Drive│      │  CLI        │      │ /Music/         │    │
│   └─────────┘      └─────────────┘      │   Artist/       │    │
│                           │             │     Album/      │    │
│                           ▼             │       01-Track  │    │
│                    ┌─────────────┐      └─────────────────┘    │
│                    │ MusicBrainz │                              │
│                    │ (metadata)  │                              │
│                    └─────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ (move HDD)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        STREAMING SERVER                         │
│                     (Pironman 5 / Pi 5)                         │
│                                                                 │
│   ┌─────────────────┐      ┌─────────────┐                     │
│   │ External HDD    │ ──▶  │  Navidrome  │ ◀── Web UI          │
│   │ /Music/         │      │  (Docker)   │ ◀── Mobile Apps     │
│   └─────────────────┘      └─────────────┘ ◀── Bluesound       │
│                                   │                             │
│                                   ▼                             │
│                            ┌─────────────┐                      │
│                            │ DLNA/UPnP   │                      │
│                            │ (Bluesound) │                      │
│                            └─────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Part 1: Ripper CLI

### Tech Stack

- **Language**: Go
- **CD Ripping**: libcdio or cdparanoia (via cgo or shell exec)
- **Audio Encoding**: FLAC encoder
- **Metadata**: MusicBrainz API
- **Album Art**: Cover Art Archive (via MusicBrainz)
- **Distribution**: GoReleaser (Windows .exe)

### Commands

```bash
# Rip a CD (interactive)
letitrip

# Rip with explicit output directory
letitrip --output D:/Music

# Rip without confirmation prompts
letitrip --yes

# Show disc info without ripping
letitrip --info

# Set default output directory
letitrip config --output D:/Music
```

### Ripping Flow

1. Detect CD in drive
2. Read disc TOC (table of contents)
3. Calculate disc ID for MusicBrainz lookup
4. Fetch metadata: artist, album, year, tracks, album art
5. Display metadata, prompt for confirmation
6. Rip each track using cdparanoia/libcdio (accurate ripping)
7. Encode to FLAC
8. Embed metadata + album art in FLAC tags
9. Save to: `{output}/{Artist}/{Album}/{TrackNum} - {Title}.flac`
10. Eject disc

### Output Structure

```
D:/Music/
├── Beatles, The/
│   ├── Abbey Road/
│   │   ├── cover.jpg
│   │   ├── 01 - Come Together.flac
│   │   ├── 02 - Something.flac
│   │   └── ...
│   └── Revolver/
│       └── ...
├── Pink Floyd/
│   └── Dark Side of the Moon/
│       └── ...
└── ...
```

### Metadata Tags (FLAC Vorbis Comments)

- `ARTIST` - Artist name
- `ALBUM` - Album name
- `TITLE` - Track title
- `TRACKNUMBER` - Track number (zero-padded)
- `TRACKTOTAL` - Total tracks
- `DATE` - Release year
- `GENRE` - Genre (if available)
- `ALBUMARTIST` - Album artist (for compilations)
- Embedded `PICTURE` - Album art (front cover)

### Windows Dependencies

The installer will bundle or install:
- FLAC encoder (flac.exe)
- CD ripping backend (TBD: libcdio/cdparanoia port or native Windows solution)

### Configuration

Config stored at: `%APPDATA%/letitrip/config.json`

```json
{
  "output_dir": "D:/Music",
  "eject_on_complete": true,
  "naming": {
    "artist_format": "last, first",  // "The Beatles" → "Beatles, The"
    "folder_template": "{artist}/{album}",
    "file_template": "{track} - {title}"
  }
}
```

---

## Part 2: Streaming Server

### Tech Stack

- **Hardware**: Raspberry Pi 5 (Pironman 5 case)
- **OS**: Raspberry Pi OS Lite (64-bit)
- **Container Runtime**: Docker
- **Music Server**: Navidrome
- **DLNA/UPnP**: Built into Navidrome or separate upmpdcli if needed

### Why Navidrome?

- Lightweight (runs great on Pi)
- Web UI for browser access
- Subsonic API compatible (mobile apps)
- Handles large libraries
- Active development
- Free and open source

### Server Setup

```bash
# One-line install on Pi
curl -fsSL https://raw.githubusercontent.com/cloudboy-jh/letitrip/main/server/setup.sh | bash
```

### Docker Compose

```yaml
version: "3"
services:
  navidrome:
    image: deluan/navidrome:latest
    container_name: navidrome
    ports:
      - "4533:4533"
    environment:
      ND_SCANSCHEDULE: 1h
      ND_LOGLEVEL: info
      ND_SESSIONTIMEOUT: 24h
      ND_BASEURL: ""
    volumes:
      - /mnt/music:/music:ro
      - navidrome-data:/data
    restart: unless-stopped

volumes:
  navidrome-data:
```

### Directory Structure (Pi)

```
/mnt/music/          # Mount point for external HDD
├── Beatles, The/
├── Pink Floyd/
└── ...

/opt/letitrip/       # Server config
├── docker-compose.yml
└── .env
```

### Access Points

- **Web UI**: `http://pironman.local:4533` or `http://<pi-ip>:4533`
- **Mobile Apps** (Subsonic compatible):
  - iOS: play:Sub, Substreamer
  - Android: Symfonium, Ultrasonic, Subtracks
- **Bluesound**: Add as DLNA/UPnP source or use Subsonic integration

### Bluesound Integration

Option A: DLNA/UPnP (simplest)
- Navidrome exposes DLNA server
- Bluesound app discovers it automatically
- Browse and play from Bluesound app

Option B: Direct Subsonic
- Some Bluesound devices support Subsonic servers directly
- Add Navidrome as a music service in Bluesound settings

---

## Repository Structure

```
letitrip/
├── README.md
├── SPEC.md
├── LICENSE
├── .github/
│   └── workflows/
│       └── release.yml       # GoReleaser automation
├── ripper/
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── internal/
│   │   ├── disc/
│   │   │   ├── detect.go     # CD detection
│   │   │   ├── read.go       # TOC reading
│   │   │   └── rip.go        # Audio extraction
│   │   ├── metadata/
│   │   │   ├── musicbrainz.go
│   │   │   └── coverart.go
│   │   ├── encode/
│   │   │   └── flac.go
│   │   ├── organize/
│   │   │   └── files.go
│   │   └── config/
│   │       └── config.go
│   ├── cmd/
│   │   └── root.go           # CLI commands (cobra)
│   └── .goreleaser.yaml
├── server/
│   ├── README.md
│   ├── docker-compose.yml
│   ├── setup.sh              # Pi setup script
│   └── navidrome.conf        # Example config
├── docs/
│   ├── ripping-guide.md
│   ├── server-setup.md
│   ├── bluesound.md
│   └── troubleshooting.md
└── install.ps1               # Windows installer script
```

---

## Installation

### Ripper (Windows)

**Option A: PowerShell installer**
```powershell
irm https://raw.githubusercontent.com/cloudboy-jh/letitrip/main/install.ps1 | iex
```

**Option B: Manual download**
1. Go to Releases page
2. Download `letitrip_windows_amd64.zip`
3. Extract to a folder in your PATH
4. Run `letitrip` from terminal

### Server (Raspberry Pi)

```bash
# SSH into your Pi, then:
curl -fsSL https://raw.githubusercontent.com/cloudboy-jh/letitrip/main/server/setup.sh | bash

# Mount your external HDD
sudo mount /dev/sda1 /mnt/music

# Start the server
cd /opt/letitrip && docker compose up -d
```

---

## User Flow

### Ripping CDs

1. Connect LG USB drive to Windows PC
2. Insert CD
3. Open terminal, run `letitrip`
4. Confirm metadata looks correct
5. Wait for rip to complete (~5-10 min per CD)
6. Disc ejects, insert next CD
7. Repeat until collection is ripped

### Setting Up Server

1. Connect external HDD to Pi
2. Run setup script
3. Access Navidrome web UI
4. Create account
5. Library auto-scans

### Listening

- **Phone**: Download Symfonium/Finamp, connect to Navidrome
- **Browser**: Go to `http://pironman.local:4533`
- **Bluesound**: Add DLNA source, browse library

---

## Future Enhancements (v2+)

- [ ] Mac support for ripper
- [ ] Linux support for ripper  
- [ ] Batch ripping mode (queue multiple discs)
- [ ] Duplicate detection
- [ ] Manual metadata editor (when MusicBrainz fails)
- [ ] Transcoding options (MP3/AAC for mobile)
- [ ] Remote access via Tailscale/Cloudflare tunnel
- [ ] iOS Shortcut / Android widget for quick ripping status
- [ ] Discord/Slack notifications on rip complete

---

## Tech Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Ripper language | Go | Cross-compile, single binary, familiar from Pact/Churn |
| Audio format | FLAC | Lossless, widely supported, good metadata support |
| Metadata source | MusicBrainz | Free, comprehensive, accurate disc ID matching |
| Music server | Navidrome | Lightweight, Pi-friendly, great mobile app ecosystem |
| Container runtime | Docker | Easy deployment, reproducible, simple updates |
| Bluesound integration | DLNA/UPnP | Native support, no extra config |

---

## References

- [MusicBrainz API](https://musicbrainz.org/doc/MusicBrainz_API)
- [Cover Art Archive](https://coverartarchive.org/)
- [Navidrome](https://www.navidrome.org/)
- [FLAC Format](https://xiph.org/flac/)
- [Pironman 5](https://docs.sunfounder.com/projects/pironman5/en/latest/)
- [Bluesound](https://www.bluesound.com/)
