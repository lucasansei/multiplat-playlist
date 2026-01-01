# multiplat-playlist

A CLI music player written in Go that seamlessly plays songs from Spotify or YouTube without switching apps.

## Why?

Sometimes you want to listen to a song only available on YouTube, but you're vibing on Spotify. Or vice versa. This tool lets you queue up songs from any platform and play them in one unified flow.

## Features

- Play Spotify and YouTube songs from the command line
- Seamless transitions between platforms
- Single interface, no app switching
- Queue songs from different platforms together

## Installation

### From source:

```bash
git clone https://github.com/yourusername/multiplat-playlist.git
cd multiplat-playlist
go build -o mplay ./cmd/mplay
sudo mv mplay /usr/local/bin/
```

### With Go install:

```bash
go install github.com/yourusername/multiplat-playlist/cmd/mplay@latest
```

## Prerequisites

- **mpv** or **ffplay** installed for audio playback
- **yt-dlp** for YouTube stream extraction

```bash
# macOS
brew install mpv yt-dlp

# Linux (Debian/Ubuntu)
sudo apt install mpv yt-dlp

# Arch Linux
sudo pacman -S mpv yt-dlp
```

## Usage

Play a single song:
```bash
mplay https://open.spotify.com/track/3n3Ppam7vgaVa1iaRUc9Lp
mplay https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

Queue multiple songs:
```bash
mplay queue add https://open.spotify.com/track/[id]
mplay queue add https://youtu.be/[video-id]
mplay queue play
```

Controls:
```bash
mplay pause
mplay resume
mplay next
mplay stop
mplay status
```

## Supported Platforms

- **Spotify** - Requires Spotify Premium and authentication
- **YouTube** - Works without authentication

### Supported URL Formats

**Spotify:**
- `https://open.spotify.com/track/[track-id]`
- `spotify:track:[track-id]`

**YouTube:**
- `https://www.youtube.com/watch?v=[video-id]`
- `https://youtu.be/[video-id]`

## Configuration

First run will prompt for Spotify credentials:
```bash
mplay auth
```

Or manually create `~/.config/multiplat-playlist/config.json`:
```json
{
  "spotify": {
    "client_id": "your_client_id",
    "client_secret": "your_client_secret"
  },
  "player": {
    "backend": "mpv",
    "volume": 100
  }
}
```

Get Spotify credentials from the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).

## Tech Stack

- **Go** - Fast, single-binary distribution, excellent concurrency
- Spotify Web API
- yt-dlp for YouTube stream extraction
- mpv for audio playback

## How It Works

1. Parse the URL to detect platform (Spotify/YouTube)
2. Fetch playback URL/stream from the respective service
3. Route to mpv player backend
4. Maintain queue state across platforms

## Project Structure

```
multiplat-playlist/
├── cmd/
│   └── mplay/           # Main CLI entry point
├── internal/
│   ├── player/          # Audio player wrapper (mpv)
│   ├── spotify/         # Spotify API client & logic
│   ├── youtube/         # YouTube stream extraction
│   ├── queue/           # Queue management
│   ├── config/          # Configuration handling
│   └── parser/          # URL parsing & detection
├── go.mod
├── go.sum
└── README.md
```

## Limitations

- Spotify playback requires Spotify Premium
- YouTube playback quality depends on available streams
- Terminal-based, no GUI controls
- Requires external dependencies (mpv, yt-dlp)

## Roadmap

- [ ] Playlist URL support
- [ ] Search functionality (play by song name)
- [ ] Save/load queues
- [ ] Interactive TUI mode with progress bar
- [ ] Cross-compilation releases (Linux, macOS, Windows)

## Contributing

Pull requests welcome. For major changes, open an issue first.

## License

MIT